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
	// OperatorPipelinesRelease is the Operator Pipelines release (version) to install.
	OperatorPipelinesRelease string `json:"operatorPipelinesRelease,omitempty"`

	// GitHubSecretName is the name of the secret containing the GitHub Token that will be used by the pipeline.
	//+kubebuilder:validation:Optional
	GitHubSecretName string `json:"gitHubSecretName,omitempty"`

	// KubeconfigSecretName is the name of the secret containing the kubeconfig that will be used by the pipeline.
	KubeconfigSecretName string `json:"kubeconfigSecretName,omitempty"`

	// The name of the secret containing the pyxis api secret expected by the pipeline
	PyxisSecretName string `json:"pyxisSecretName,omitempty"`

	// The name of the secret containing the docker registry credentials secret expected by the pipeline
	DockerRegistrySecretName string `json:"dockerRegistrySecretName,omitempty"`

	// The name of the secret containing the github ssh secret expected by the pipeline
	GithubSSHSecretName string `json:"githubSSHSecretName,omitempty"`

	// ApplyCIPipeline determines whether to install the ci pipeline.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="CI Pipeline",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	//+kubebuilder:validation:Required
	ApplyCIPipeline bool `json:"applyCIPipeline"`

	// ApplyHostedPipeline determines whether to install the hosted pipeline.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Hosted Pipeline",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	//+kubebuilder:validation:Required
	ApplyHostedPipeline bool `json:"applyHostedPipeline"`

	// ApplyReleasePipeline determines whether to install the release pipeline.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Release Pipeline",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	//+kubebuilder:validation:Required
	ApplyReleasePipeline bool `json:"applyReleasePipeline"`
}

// OperatorPipelineStatus defines the observed state of OperatorPipeline
type OperatorPipelineStatus struct {
	// conditions describes the state of the operator's reconciliation functionality.
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +optional
	// Conditions is a list of conditions related to operator reconciliation
	Conditions []metav1.Condition `json:"conditions,omitempty"  patchStrategy:"merge" patchMergeKey:"type"`

	// ObservedGeneration is the generation last observed by the controller
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
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
