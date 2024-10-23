package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LocustTestSpec defines the desired state of LocustTest
type LocustTestSpec struct {
	// Image to use for running Locust tests
	Image string `json:"image"`

	// Number of users to simulate in the Locust test
	Users int32 `json:"users"`

	// Rate at which users will be spawned (users per second)
	SpawnRate int32 `json:"spawnRate"`

	// Duration for which the load test will run (e.g., "5m", "10m")
	RunTime string `json:"runTime"`

	// The URL of the target application to load test
	TargetURL string `json:"targetURL"`

	// Number of worker pods to run (each pod simulates part of the load)
	Workers int32 `json:"workers"`

	// Optional: Resource limits/requests for master and worker pods
	// MasterResources defines resource requests/limits for the master pod
	MasterResources *corev1.ResourceRequirements `json:"masterResources,omitempty"`

	// WorkerResources defines resource requests/limits for the worker pods
	WorkerResources *corev1.ResourceRequirements `json:"workerResources,omitempty"`

	// Optional: Start time for the load test to begin
	StartTime *metav1.Time `json:"startTime,omitempty"`
}

// LocustTestStatus defines the observed state of LocustTest
type LocustTestStatus struct {
	// Conditions to reflect the status of the Locust test (e.g., "Running", "Completed", "Failed")
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Total number of users currently spawned
	SpawnedUsers int32 `json:"spawnedUsers,omitempty"`

	// Timestamp when the test started
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// Timestamp when the test finished
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// LocustTest is the Schema for the locusttests API
type LocustTest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LocustTestSpec   `json:"spec,omitempty"`
	Status LocustTestStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// LocustTestList contains a list of LocustTest
type LocustTestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LocustTest `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LocustTest{}, &LocustTestList{})
}
