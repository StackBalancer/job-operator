package v1

import "k8s.io/apimachinery/pkg/runtime"

// DeepCopyInto copies all properties of this object into another object of the
// same type that is provided as a pointer.
func (in *TaskJob) DeepCopyInto(out *TaskJob) {
	out.TypeMeta = in.TypeMeta
	out.ObjectMeta = in.ObjectMeta
	out.Spec = TaskJobSpec{
		JobName:   in.Spec.JobName,
		JobParams: in.Spec.JobParams,
		Image:     in.Spec.Image,
		ImagePullPolicy: in.Spec.ImagePullPolicy,
		Replicas:  in.Spec.Replicas,
	}
}

// DeepCopyObject returns a generically typed copy of an object
func (in *TaskJob) DeepCopyObject() runtime.Object {
	out := TaskJob{}
	in.DeepCopyInto(&out)

	return &out
}

// DeepCopyObject returns a generically typed copy of an object
func (in *TaskJobList) DeepCopyObject() runtime.Object {
	out := TaskJobList{}
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta

	if in.Items != nil {
		out.Items = make([]TaskJob, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}

	return &out
}
