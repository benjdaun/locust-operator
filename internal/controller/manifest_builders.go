package controller

import (
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
	testsv1alpha1 "load.test/locust/api/v1alpha1" // Adjust according to your project structure
)

func ConstructMasterJob(locustTest *testsv1alpha1.LocustTest) *batchv1.Job {
	labels := map[string]string{
		"app":        "locust-master",
		"locusttest": locustTest.Name,
	}

	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      locustTest.Name + "-master",
			Namespace: locustTest.Namespace,
			Labels:    labels,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:    "locust-master",
							Image:   locustTest.Spec.Image,
							Command: []string{"locust"},
							Args: []string{
								"--master", // Run as master
								"--headless",
								"-u", fmt.Sprintf("%d", locustTest.Spec.Users),
								"-r", fmt.Sprintf("%d", locustTest.Spec.SpawnRate),
								"--run-time", locustTest.Spec.RunTime,
								"--host", locustTest.Spec.TargetURL,
								"--expect-workers", fmt.Sprintf("%d", locustTest.Spec.Workers),
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyNever, // Ensure the job only runs once
				},
			},
		},
	}
}

func ConstructWorkerJob(locustTest *testsv1alpha1.LocustTest, workerIndex int) *batchv1.Job {
	labels := map[string]string{
		"app":        "locust-worker",
		"locusttest": locustTest.Name,
	}
	ttlAfterFinished := int32(30)

	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-worker-%d", locustTest.Name, workerIndex), // Unique name for each worker
			Namespace: locustTest.Namespace,
			Labels:    labels,
		},
		Spec: batchv1.JobSpec{
			TTLSecondsAfterFinished: &ttlAfterFinished,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:    "locust-worker",
							Image:   locustTest.Spec.Image,
							Command: []string{"locust"},
							Args: []string{
								"--worker", // Run as worker
								fmt.Sprintf("--master-host=%s-master", locustTest.Name), // Connect to master via service
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyNever, // Ensure the worker job only runs once
				},
			},
		},
	}
}

func ConstructMasterService(locustTest *testsv1alpha1.LocustTest) *corev1.Service {
	labels := map[string]string{
		"app":        "locust-master",
		"locusttest": locustTest.Name,
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      locustTest.Name + "-master",
			Namespace: locustTest.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Port:       5557,
					TargetPort: intstr.FromInt(5557),
				},
			},
		},
	}
}
