package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DatabaseSpec defines the desired state of Database
type DatabaseSpec struct {
	// Name of the database (also used as StatefulSet name)
	DatabaseName string `json:"databaseName"`
	// Postgres image, e.g. postgres:15-alpine
	Image string `json:"image"`
	// number of replicas (1 is typical for single-primary demo)
	Replicas int `json:"replicas"`
	// Storage size e.g. "1Gi"
	Storage string `json:"storage,omitempty"`
	// Postgres password (for demo; in real world use Secrets)
	Password string `json:"password,omitempty"`
	// Optional image pull policy (IfNotPresent/Always)
	ImagePullPolicy string `json:"imagePullPolicy,omitempty"`
}

// DatabaseStatus defines the observed state of Database
type DatabaseStatus struct {
	// Phase is one of Pending/Running/Ready/Failed
	Phase string `json:"phase,omitempty"`
	// Number of ready replicas
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`
	// Conditions, optional in future
}

// Database is the Schema for the Database Custom Resource
type Database struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DatabaseSpec   `json:"spec,omitempty"`
	Status DatabaseStatus `json:"status,omitempty"`
}

type DatabaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Database `json:"items"`
}
