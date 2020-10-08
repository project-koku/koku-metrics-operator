// +build !ignore_autogenerated

/*


Copyright 2020 Red Hat, Inc.

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AuthenticationSpec) DeepCopyInto(out *AuthenticationSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AuthenticationSpec.
func (in *AuthenticationSpec) DeepCopy() *AuthenticationSpec {
	if in == nil {
		return nil
	}
	out := new(AuthenticationSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AuthenticationStatus) DeepCopyInto(out *AuthenticationStatus) {
	*out = *in
	if in.AuthenticationCredentialsFound != nil {
		in, out := &in.AuthenticationCredentialsFound, &out.AuthenticationCredentialsFound
		*out = new(bool)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AuthenticationStatus.
func (in *AuthenticationStatus) DeepCopy() *AuthenticationStatus {
	if in == nil {
		return nil
	}
	out := new(AuthenticationStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CloudDotRedHatSourceSpec) DeepCopyInto(out *CloudDotRedHatSourceSpec) {
	*out = *in
	if in.CreateSource != nil {
		in, out := &in.CreateSource, &out.CreateSource
		*out = new(bool)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CloudDotRedHatSourceSpec.
func (in *CloudDotRedHatSourceSpec) DeepCopy() *CloudDotRedHatSourceSpec {
	if in == nil {
		return nil
	}
	out := new(CloudDotRedHatSourceSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CloudDotRedHatSourceStatus) DeepCopyInto(out *CloudDotRedHatSourceStatus) {
	*out = *in
	if in.SourceDefined != nil {
		in, out := &in.SourceDefined, &out.SourceDefined
		*out = new(bool)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CloudDotRedHatSourceStatus.
func (in *CloudDotRedHatSourceStatus) DeepCopy() *CloudDotRedHatSourceStatus {
	if in == nil {
		return nil
	}
	out := new(CloudDotRedHatSourceStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CostManagement) DeepCopyInto(out *CostManagement) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CostManagement.
func (in *CostManagement) DeepCopy() *CostManagement {
	if in == nil {
		return nil
	}
	out := new(CostManagement)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *CostManagement) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CostManagementList) DeepCopyInto(out *CostManagementList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]CostManagement, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CostManagementList.
func (in *CostManagementList) DeepCopy() *CostManagementList {
	if in == nil {
		return nil
	}
	out := new(CostManagementList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *CostManagementList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CostManagementSpec) DeepCopyInto(out *CostManagementSpec) {
	*out = *in
	if in.ValidateCert != nil {
		in, out := &in.ValidateCert, &out.ValidateCert
		*out = new(bool)
		**out = **in
	}
	out.Authentication = in.Authentication
	in.Upload.DeepCopyInto(&out.Upload)
	in.Source.DeepCopyInto(&out.Source)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CostManagementSpec.
func (in *CostManagementSpec) DeepCopy() *CostManagementSpec {
	if in == nil {
		return nil
	}
	out := new(CostManagementSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CostManagementStatus) DeepCopyInto(out *CostManagementStatus) {
	*out = *in
	if in.ValidateCert != nil {
		in, out := &in.ValidateCert, &out.ValidateCert
		*out = new(bool)
		**out = **in
	}
	in.Authentication.DeepCopyInto(&out.Authentication)
	in.Upload.DeepCopyInto(&out.Upload)
	in.Source.DeepCopyInto(&out.Source)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CostManagementStatus.
func (in *CostManagementStatus) DeepCopy() *CostManagementStatus {
	if in == nil {
		return nil
	}
	out := new(CostManagementStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *UploadSpec) DeepCopyInto(out *UploadSpec) {
	*out = *in
	if in.UploadWait != nil {
		in, out := &in.UploadWait, &out.UploadWait
		*out = new(int64)
		**out = **in
	}
	if in.UploadCycle != nil {
		in, out := &in.UploadCycle, &out.UploadCycle
		*out = new(int64)
		**out = **in
	}
	if in.UploadToggle != nil {
		in, out := &in.UploadToggle, &out.UploadToggle
		*out = new(bool)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new UploadSpec.
func (in *UploadSpec) DeepCopy() *UploadSpec {
	if in == nil {
		return nil
	}
	out := new(UploadSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *UploadStatus) DeepCopyInto(out *UploadStatus) {
	*out = *in
	if in.UploadToggle != nil {
		in, out := &in.UploadToggle, &out.UploadToggle
		*out = new(bool)
		**out = **in
	}
	if in.UploadWait != nil {
		in, out := &in.UploadWait, &out.UploadWait
		*out = new(int64)
		**out = **in
	}
	if in.UploadCycle != nil {
		in, out := &in.UploadCycle, &out.UploadCycle
		*out = new(int64)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new UploadStatus.
func (in *UploadStatus) DeepCopy() *UploadStatus {
	if in == nil {
		return nil
	}
	out := new(UploadStatus)
	in.DeepCopyInto(out)
	return out
}
