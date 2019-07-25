package v1

import (
	"github.com/knative/pkg/apis/duck/v1alpha1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type PipelineRun struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PipelineRunSpec   `json:"spec,omitempty"`
	Status PipelineRunStatus `json:"status,omitempty"`
}

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

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

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
