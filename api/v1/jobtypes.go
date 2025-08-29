package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TaskJobSpec defines the desired state of TaskJob
type TaskJobSpec struct {
	JobName         string            `json:"jobName"`
	JobParams       map[string]string `json:"jobParams,omitempty"` // Optional job parameters as key-value pairs
	Image           string            `json:"image"`
	ImagePullPolicy string            `json:"imagePullPolicy,omitempty"`
	Replicas        int               `json:"replicas"`
}

// TaskJobStatus defines the observed state of TaskJob
type TaskJobStatus struct {
	State           string      `json:"state"` // State of the job (Pending, Running, Completed, Failed)                    // Current state of the TaskJob
	CompletionTime *metav1.Time `json:"completionTime,omitempty"` // Optional completion timestamp
}

// TaskJob is the Schema for the TaskJob Custom Resource
type TaskJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TaskJobSpec   `json:"spec,omitempty"`
	Status TaskJobStatus `json:"status,omitempty"`
}

// TaskJobList contains a list of TaskJob
type TaskJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TaskJob `json:"items"`
}
