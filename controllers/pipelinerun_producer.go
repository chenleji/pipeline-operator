package controllers

import (
	"context"
	"github.com/astaxie/beego/logs"
	"github.com/chenleji/pipeline-operator/api/v1"
	"github.com/go-logr/logr"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PipelineRunProducer struct {
	logger     logr.Logger
	client     client.Client
	kubeClient *kubernetes.Clientset
	stopCh     <-chan struct{}
}

func NewPipelineRunProducer(logger logr.Logger,
	client client.Client,
	kubeClient *kubernetes.Clientset,
	stopCh <-chan struct{}) *PipelineRunProducer {

	return &PipelineRunProducer{
		logger:     logger,
		client:     client,
		kubeClient: kubeClient,
		stopCh:     stopCh,
	}
}

func (p *PipelineRunProducer) Handler() {
	for {
		select {
		case <-p.stopCh:
			return

		case run := <-v1.TodoPipelineRunChan:
			if err := p.client.Create(context.Background(), run); err != nil {
				// retry again
				v1.TodoPipelineRunChan <- run
			}

			logs.Debug("[pipeline] cronTask create pipelineRun success!", run.Namespace, run.Name)
		}
	}
}
