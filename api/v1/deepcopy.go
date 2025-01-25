package v1

import "k8s.io/apimachinery/pkg/runtime"

// DeepCopyInto copies all properties of this object into another object of the
// same type that is provided as a pointer.
func (in *HPCJob) DeepCopyInto(out *HPCJob) {
	out.TypeMeta = in.TypeMeta
	out.ObjectMeta = in.ObjectMeta
	out.Spec = HPCJobSpec{
		JobName:   in.Spec.JobName,
		State:     in.Spec.State,
		JobParams: in.Spec.JobParams,
		Replicas:  in.Spec.Replicas,
		Image:     in.Spec.Image,
	}
}

// DeepCopyObject returns a generically typed copy of an object
func (in *HPCJob) DeepCopyObject() runtime.Object {
	out := HPCJob{}
	in.DeepCopyInto(&out)

	return &out
}

// DeepCopyObject returns a generically typed copy of an object
func (in *HPCJobList) DeepCopyObject() runtime.Object {
	out := HPCJobList{}
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta

	if in.Items != nil {
		out.Items = make([]HPCJob, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}

	return &out
}
