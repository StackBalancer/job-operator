package v1

import "k8s.io/apimachinery/pkg/runtime"

// DeepCopyInto copies all properties of this object into another object of the
// same type that is provided as a pointer.
func (in *Database) DeepCopyInto(out *Database) {
	out.TypeMeta = in.TypeMeta
	out.ObjectMeta = in.ObjectMeta
	out.Spec = DatabaseSpec{
		DatabaseName:    in.Spec.DatabaseName,
		Image:           in.Spec.Image,
		Replicas:        in.Spec.Replicas,
		Storage:         in.Spec.Storage,
		Password:        in.Spec.Password,
		ImagePullPolicy: in.Spec.ImagePullPolicy,
	}
}

// DeepCopyObject returns a generically typed copy of an object
func (in *Database) DeepCopyObject() runtime.Object {
	out := Database{}
	in.DeepCopyInto(&out)

	return &out
}

// DeepCopyObject returns a generically typed copy of an object
func (in *DatabaseList) DeepCopyObject() runtime.Object {
	out := DatabaseList{}
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta

	if in.Items != nil {
		out.Items = make([]Database, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}

	return &out
}
