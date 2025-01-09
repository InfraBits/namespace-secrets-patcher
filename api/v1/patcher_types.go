/*
Copyright 2025 Infra Bits.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

type TargetSpec struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// PatcherSpec defines the desired state of Patcher.
type PatcherSpec struct {
	Secret  string       `json:"secret"`
	Targets []TargetSpec `json:"targets"`
}

func (ps *Patcher) NameSpaceIsTarget(namespace string) bool {
	for _, target := range ps.Spec.Targets {
		if (target.Type == "prefix" && strings.HasPrefix(namespace, target.Name)) || (target.Type == "match" && namespace == target.Name) {
			return true
		}
	}
	return false
}

// PatcherStatus defines the observed state of Patcher.
type PatcherStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Patcher is the Schema for the patchers API.
type Patcher struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PatcherSpec   `json:"spec,omitempty"`
	Status PatcherStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PatcherList contains a list of Patcher.
type PatcherList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Patcher `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Patcher{}, &PatcherList{})
}
