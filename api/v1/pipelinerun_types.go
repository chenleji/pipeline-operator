/*

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
	"github.com/knative/pkg/apis/duck/v1alpha1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PipelineRunSpec defines the desired state of PipelineRun
type PipelineRunSpec struct {
	RefPipeline string `json:"refPipeline"`
	// +optional
	MaxRetryCount int32 `json:"maxRetryCount,omitempty"`
	// +optional
	Timeout *metav1.Duration `json:"timeout,omitempty"`
}

// PipelineRunStatus defines the observed state of PipelineRun
type PipelineRunStatus struct {
	v1alpha1.Status `json:",inline"`
	// +optional
	RetryStatus *RetryStatus `json:"retryStatus,omitempty"`
	// +optional
	Cluster *ClusterSpec `json:"cluster,omitempty"`
	// +optional
	LastJobStatus string `json:"lastJobStatus,omitempty"`
	// +optional
	IdleStatus string `json:"idleStatus,omitempty"`
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`
	// +optional
	StepStatus []v1.ContainerState `json:"stepStates,omitempty"`
	// +optional
	StepsCompleted []string `json:"stepsCompleted,omitempty"`
}

// +kubebuilder:object:root=true

// PipelineRun is the Schema for the pipelineruns API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type PipelineRun struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PipelineRunSpec   `json:"spec,omitempty"`
	Status PipelineRunStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PipelineRunList contains a list of PipelineRun
type PipelineRunList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PipelineRun `json:"items"`
}

type ClusterSpec struct {
	Namespace       string   `json:"namespace"`
	BuildName       string   `json:"buildName,omitempty"`
	DeploymentNames []string `json:"deploymentNames,omitempty"`
	ServiceNames    []string `json:"serviceNames,omitempty"`
}

type RetryStatus struct {
	RetryCount   int32         `json:"retryCount"`
	HistoryBuild []ClusterSpec `json:"historyBuild,omitempty"`
}

var pipelineRunCondSet = v1alpha1.NewBatchConditionSet()

func (pr *PipelineRunStatus) GetCondition(t v1alpha1.ConditionType) *v1alpha1.Condition {
	return pipelineRunCondSet.Manage(pr).GetCondition(t)
}

func (pr *PipelineRunStatus) InitializeConditions() {
	if pr.StartTime.IsZero() {
		pr.StartTime = &metav1.Time{Time: time.Now()}
	}
	pipelineRunCondSet.Manage(pr).InitializeConditions()
}

func (pr *PipelineRunStatus) SetCondition(newCond *v1alpha1.Condition) {
	if newCond != nil {
		pipelineRunCondSet.Manage(pr).SetCondition(*newCond)
	}
}

func (pr *PipelineRunStatus) SetConditions(newConds v1alpha1.Conditions) {
	if len(newConds) != 0 {
		for _, newCond := range newConds {
			pipelineRunCondSet.Manage(pr).SetCondition(newCond)
		}
	}
}

func init() {
	SchemeBuilder.Register(&PipelineRun{}, &PipelineRunList{})
}
