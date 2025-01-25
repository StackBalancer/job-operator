package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// API group name and version of custom resource
const GroupName = "kubernetes.hpc.com"
const GroupVersion = "v1"

var SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: GroupVersion}

var (
	// SchemeBuilder helps to register the types with the scheme
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	// AddToScheme adds CRD types to the scheme
	AddToScheme = SchemeBuilder.AddToScheme
)

// addKnownTypes registers the custom resource types in the Scheme
func addKnownTypes(scheme *runtime.Scheme) error {
	// Register the resource and its list type HPCJob and HPCJobList
	scheme.AddKnownTypes(SchemeGroupVersion,
		&HPCJob{},
		&HPCJobList{},
	)

	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
