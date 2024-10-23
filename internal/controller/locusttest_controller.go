package controller

import (
	"context"
	"fmt"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	testsv1alpha1 "load.test/locust/api/v1alpha1"
)

// Required RBAC permissions for the controller

// Grant access to Jobs
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete

// Grant access to Services
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete

// Grant access to updating the status of LocustTest resources
// +kubebuilder:rbac:groups=tests.load.test,resources=locusttests/status,verbs=get;update;patch

// Grant access to create and update finalizers (if needed)
// +kubebuilder:rbac:groups=tests.load.test,resources=locusttests/finalizers,verbs=update

// LocustTestReconciler reconciles a LocustTest object
type LocustTestReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *LocustTestReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	// Fetch the LocustTest instance
	locustTest := &testsv1alpha1.LocustTest{}
	if err := r.Get(ctx, req.NamespacedName, locustTest); err != nil {
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	// If the test is already completed, no need to recreate jobs
	if conditionExists(locustTest, "Completed") {
		fmt.Println("Test already completed. Skipping job creation.")
		return reconcile.Result{}, nil
	}

	// Check if the test has a start time and if the current time is before that start time
	if locustTest.Spec.StartTime != nil && time.Now().Before(locustTest.Spec.StartTime.Time) {
		// Log the fact that it's not time to start the test yet
		fmt.Println("Test start time is in the future, requeueing until the start time.")
		// Requeue until the start time
		return reconcile.Result{RequeueAfter: time.Until(locustTest.Spec.StartTime.Time)}, nil
	}

	masterJobName := fmt.Sprintf("%s-master", locustTest.Name)
	masterJob := &batchv1.Job{}
	err := r.Get(ctx, client.ObjectKey{Namespace: locustTest.Namespace, Name: masterJobName}, masterJob)
	// If the error is not a "not found" error, handle it
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			// Return the error if it's not a "not found" error
			return reconcile.Result{}, err
		}
		// If "not found", proceed to create the job
	} else {
		// If the master job is found and completed, check job completion and clean up
		jobCompleted, err := r.CheckJobCompletion(ctx, masterJob, locustTest)
		if err != nil {
			return reconcile.Result{}, err
		}

		// If the job has completed, clean up the master job and service
		if jobCompleted {
			if err := r.CleanUpMaster(ctx, locustTest); err != nil {
				return reconcile.Result{}, err
			}
			// Job completed, no need to requeue
			return reconcile.Result{}, nil
		}
	}
	// Create or update the master service
	masterService := ConstructMasterService(locustTest)
	if err := r.CreateOrUpdateService(ctx, masterService, locustTest); err != nil {
		return reconcile.Result{}, err
	}

	// Create or update the master job
	masterJob = ConstructMasterJob(locustTest)
	if err := r.CreateOrUpdateJob(ctx, masterJob, locustTest); err != nil {
		return reconcile.Result{}, err
	}

	// Create multiple worker jobs based on the Workers
	for i := 0; i < int(locustTest.Spec.Workers); i++ {
		workerJob := ConstructWorkerJob(locustTest, i) // Pass worker index for unique naming
		if err := r.CreateOrUpdateJob(ctx, workerJob, locustTest); err != nil {
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{RequeueAfter: time.Minute}, nil
}

// CreateOrUpdateJob ensures that the job exists and is managed correctly
func (r *LocustTestReconciler) CreateOrUpdateJob(ctx context.Context, job *batchv1.Job, locustTest *testsv1alpha1.LocustTest) error {
	if err := controllerutil.SetControllerReference(locustTest, job, r.Scheme); err != nil {
		return err
	}

	found := &batchv1.Job{}
	err := r.Get(ctx, client.ObjectKey{Namespace: job.Namespace, Name: job.Name}, found)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			// Job doesn't exist, so create it
			if err := r.Create(ctx, job); err != nil {
				return err
			}
		} else {
			return err
		}
	} else {
		// Job exists, no need to update it
	}
	return nil
}

// CreateOrUpdateService ensures that the service exists and is managed correctly
func (r *LocustTestReconciler) CreateOrUpdateService(ctx context.Context, service *corev1.Service, locustTest *testsv1alpha1.LocustTest) error {
	if err := controllerutil.SetControllerReference(locustTest, service, r.Scheme); err != nil {
		return err
	}

	found := &corev1.Service{}
	err := r.Get(ctx, client.ObjectKey{Namespace: service.Namespace, Name: service.Name}, found)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			// Service doesn't exist, so create it
			if err := r.Create(ctx, service); err != nil {
				return err
			}
		} else {
			return err
		}
	} else {
		// Service exists, no need to update it
	}
	return nil
}

// CheckJobCompletion checks the status of the master job and cleans up resources when it finishes
// CheckJobCompletion checks the status of the job and updates the LocustTest status if completed.
// It returns a boolean indicating whether the job has completed successfully.
func (r *LocustTestReconciler) CheckJobCompletion(ctx context.Context, job *batchv1.Job, locustTest *testsv1alpha1.LocustTest) (bool, error) {
	foundJob := &batchv1.Job{}
	err := r.Get(ctx, client.ObjectKey{Namespace: job.Namespace, Name: job.Name}, foundJob)
	if err != nil {
		return false, err
	}

	// Check if the job has completed successfully
	if foundJob.Status.Succeeded > 0 {
		// If the "Completed" condition already exists, skip updating the status again
		if !conditionExists(locustTest, "Completed") {
			locustTest.Status.Conditions = append(locustTest.Status.Conditions, metav1.Condition{
				Type:               "Completed",
				Status:             metav1.ConditionTrue,
				LastTransitionTime: metav1.Now(),
				Reason:             "LocustTestCompleted",
				Message:            "The Locust load test has completed successfully.",
			})
			if err := r.Status().Update(ctx, locustTest); err != nil {
				return false, err
			}
		}

		// Return true indicating that the job has completed
		return true, nil
	}

	// Job has not yet completed
	return false, nil
}

func (r *LocustTestReconciler) CleanUpMaster(ctx context.Context, locustTest *testsv1alpha1.LocustTest) error {
	// Find and delete the master job
	masterJobName := fmt.Sprintf("%s-master", locustTest.Name)
	masterJob := &batchv1.Job{}
	err := r.Get(ctx, client.ObjectKey{Namespace: locustTest.Namespace, Name: masterJobName}, masterJob)
	if err == nil {
		deletePolicy := metav1.DeletePropagationForeground
		if err := r.Delete(ctx, masterJob, &client.DeleteOptions{
			PropagationPolicy: &deletePolicy,
		}); err != nil {
			return fmt.Errorf("failed to delete master job: %v", err)
		}
		fmt.Printf("Master job %s deleted successfully\n", masterJobName)
	}

	// Find and delete the master service
	masterServiceName := fmt.Sprintf("%s-master", locustTest.Name)
	masterService := &corev1.Service{}
	err = r.Get(ctx, client.ObjectKey{Namespace: locustTest.Namespace, Name: masterServiceName}, masterService)
	if err == nil {
		// Delete the master service
		if err := r.Delete(ctx, masterService); err != nil {
			return fmt.Errorf("failed to delete master service: %v", err)
		}
		fmt.Printf("Master service %s deleted successfully\n", masterServiceName)
	}

	return nil
}

func conditionExists(locustTest *testsv1alpha1.LocustTest, conditionType string) bool {
	for _, condition := range locustTest.Status.Conditions {
		if condition.Type == conditionType {
			return true
		}
	}
	return false
}

// SetupWithManager sets up the controller with the Manager.
func (r *LocustTestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&testsv1alpha1.LocustTest{}).
		Complete(r)
}
