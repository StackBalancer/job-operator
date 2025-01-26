package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// HPCJobSpec defines the desired state of HPCJob
type HPCJobSpec struct {
	JobName         string            `json:"jobName"`             // Name of the HPC job
	State           string            `json:"state"`               // State of the job (Pending, Running, Completed, Failed)
	JobParams       map[string]string `json:"jobParams,omitempty"` // Optional job parameters as key-value pairs
	Image           string            `json:"image"`
	ImagePullPolicy string            `json:"imagePullPolicy,omitempty"`
	Replicas        int               `json:"replicas"`
}

// HPCJobStatus defines the observed state of HPCJob
type HPCJobStatus struct {
	State          string       `json:"state"`                    // Current state of the HPC job
	CompletionTime *metav1.Time `json:"completionTime,omitempty"` // Optional completion timestamp
}

// HPCJob is the Schema for the HPCJob Custom Resource
type HPCJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HPCJobSpec   `json:"spec,omitempty"`
	Status HPCJobStatus `json:"status,omitempty"`
}

// HPCJobList contains a list of HPCJob
type HPCJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HPCJob `json:"items"`
}
