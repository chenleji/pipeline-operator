package v1

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/astaxie/beego/logs"
	"io"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

var TodoPipelineRunChan = make(chan *PipelineRun, 20)

const (
	PipelineLabelKey     = "pipeline"
	PipelineRunFinalizer = "finalizer.ljchen.net"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Pipeline struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PipelineSpec   `json:"spec,omitempty"`
	Status PipelineStatus `json:"status,omitempty"`
}

type PipelineSpec struct {
	// +optional
	BatchJob *PipelineBatchJob `json:"batchJob,omitempty"`
	// +optional
	StreamJob *PipelineStreamJob `json:"streamJob,omitempty"`
	// +optional
	Strategy *PipelineStrategy `json:"strategy,omitempty"`
}

type PipelineStatus struct {
	Idle bool `json:"idle"`
	// +optional
	UsedPipelineRun []*UsedPipelineRun `json:"usedPipelineRun,omitempty"`
}

type PipelineBatchJob struct {
	Input *PipelineBatchJobInput `json:"input"`
	// +optional
	Processors []corev1.Container `json:"processors,omitempty"`

	Outputs []*PipelineBatchJobOutput `json:"outputs"`
	// +optional
	Volumes []corev1.Volume `json:"volumes,omitempty"`
	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty"`
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty" protobuf:"bytes,8,opt,name=resources"`
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`
}

type PipelineStreamJob struct {
	MicroServices []PipelineStreamJobMicroServices `json:"microServices"`
}

type PipelineStrategy struct {
	// CronExpression, eg: {cronExpression}
	// +optional
	CronExpression string `json:"cronExpression,omitempty"`
	// Expire time, eg: 2019-12-30 24:00:00
	Expire string `json:"expire"`
}

type PipelineBatchJobInput struct {
	// +optional
	ElasticSearch map[string]string `json:"elasticSearch,omitempty"`
	// +optional
	ApiServer map[string]string `json:"apiServer,omitempty"`
}

type PipelineBatchJobOutput struct {
	// Output stage name
	Name string `json:"name"`
	// +optional
	Api map[string]string `json:"api,omitempty"`
	// +optional
	Ftp map[string]string `json:"ftp,omitempty"`
	// +optional
	As2 map[string]string `json:"as2,omitempty"`
	// +optional
	Kafka map[string]string `json:"kafka,omitempty"`
	// +optional
	Email map[string]string `json:"email,omitempty"`
	// +optional
	Custom map[string]string `json:"custom,omitempty"`
}

type PipelineStreamJobMicroServices struct {
	Name     string         `json:"name"`
	Replicas *int32         `json:"replicas,omitempty" protobuf:"varint,1,opt,name=replicas"`
	PodSpec  corev1.PodSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
}

type UsedPipelineRun struct {
	Namespace   string `json:"namespace"`
	PipelineRun string `json:"pipelineRun"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FooList is a list of Foo resources
type PipelineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Pipeline `json:"items"`
}

func (p *Pipeline) CronTask() {
	fmt.Println("trigger ...... cron task.........")

	// check expire
	if len(p.Spec.Strategy.Expire) != 0 {
		expireTime, err := time.Parse("2006-01-02 15:04:05", p.Spec.Strategy.Expire)
		if err != nil {
			return
		}

		if !expireTime.After(time.Now()) {
			logs.Info("[pipeline] expired... ", p.Namespace, p.Name)
			return
		}
	}

	getNameFun := func(parent string) string {
		var randReader = rand.Reader
		b, err := ioutil.ReadAll(io.LimitReader(randReader, 3))
		if err != nil {
			panic("gun")
		}
		return parent + "-run-" + hex.EncodeToString(b)
	}

	TodoPipelineRunChan <- &PipelineRun{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PipelineRun",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      getNameFun(p.Name),
			Namespace: p.Namespace,
			Labels: map[string]string{
				PipelineLabelKey: p.Name,
			},
			Finalizers: []string{
				PipelineRunFinalizer,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(p.GetObjectMeta(), p.GroupVersionKind()),
			},
		},
		Spec: PipelineRunSpec{
			RefPipeline: p.Name,
		},
	}
}
