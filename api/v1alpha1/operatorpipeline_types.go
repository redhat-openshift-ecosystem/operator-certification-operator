/*
Copyright 2021.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Important: Run "make" to regenerate code after modifying this file
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// OperatorPipelineSpec defines the desired state of OperatorPipeline
type OperatorPipelineSpec struct {
	// OpenShiftPipelineVersion is the version of the OpenShift Pipelines Operator to install.
	OpenShiftPipelineVersion string `json:"openShiftPipelineVersion,omitempty"`

	// OperatorPipelinesRelease is the Operator Pipelines release (version) to install.
	OperatorPipelinesRelease string `json:"operatorPipelinesRelease,omitempty"`

	// GitHubSecretName is the name of the secret containing the GitHub Token that will be used by the pipeline.
	//+kubebuilder:validation:Optional
	GitHubSecretName string `json:"gitHubSecretName,omitempty"`

	// KubeconfigSecretName is the name of the secret containing the kubeconfig that will be used by the pipeline.
	KubeconfigSecretName string `json:"kubeconfigSecretName,omitempty"`
	
	// The name of the secret containing the pyxis api secret expected by the pipeline
	PyxisApiSecretName string `json:"pyxisApiSecretName,omitempty"`
}

// OperatorPipelineStatus defines the observed state of OperatorPipeline
type OperatorPipelineStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// OperatorPipeline is the Schema for the operatorpipelines API
type OperatorPipeline struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OperatorPipelineSpec   `json:"spec,omitempty"`
	Status OperatorPipelineStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OperatorPipelineList contains a list of OperatorPipeline
type OperatorPipelineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OperatorPipeline `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OperatorPipeline{}, &OperatorPipelineList{})
}
