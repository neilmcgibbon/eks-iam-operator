/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SyncState string

const (
	SyncStateOK  SyncState = "OK"
	SyncStateErr SyncState = "ERROR"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// RoleSpec defines the desired state of Role
type RoleSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// List of service account names in
	// +kubebuilder:validation:Required
	ServiceAccounts []string `json:"serviceAccounts"`

	// +kubebuilder:validation:Required
	Namespace string `json:"namespace"`

	// +kubebuilder:validation:Required
	Statements map[string][]StatementSpec `json:"statements"`
}

// StatementSpec defines an actual inline permission
type StatementSpec struct {

	// +kubebuilder:validation:Required
	Actions []string `json:"actions"`

	// +kubebuilder:validation:Required
	Resources []string `json:"resources"`
}

// RoleStatus defines the observed state of Role
type RoleStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	State              SyncState `json:"state"`
	Error              string    `json:"error"`
	ObservedGeneration int64     `json:"observedGeneration"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Role is the Schema for the roles API
type Role struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RoleSpec   `json:"spec,omitempty"`
	Status RoleStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RoleList contains a list of Role
type RoleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Role `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Role{}, &RoleList{})
}
