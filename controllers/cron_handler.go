package controllers

import (
	"context"
	"fmt"
	"github.com/chenleji/pipeline-operator/api/v1"
	"github.com/astaxie/beego/logs"
	"github.com/go-logr/logr"
	buildclientset "github.com/knative/build/pkg/client/clientset/versioned"
	"github.com/robfig/cron"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sync"
	"time"
)

var (
	registerMut  = sync.Mutex{}
	registerDict = make(map[string]chan bool)
)

type CronTaskSet struct {
	logger      logr.Logger
	client      client.Client
	buildClient *buildclientset.Clientset
	kubeClient  *kubernetes.Clientset
	stopCh      <-chan struct{}
}

func NewCronHandler(logger logr.Logger,
	client client.Client,
	buildClient *buildclientset.Clientset,
	kubeClient *kubernetes.Clientset,
	stopCh <-chan struct{}) *CronTaskSet {

	return &CronTaskSet{
		logger:      logger,
		client:      client,
		buildClient: buildClient,
		kubeClient:  kubeClient,
		stopCh:      stopCh,
	}
}

func (c *CronTaskSet) CheckCronTask() {
	namespaces, err := c.kubeClient.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		c.logger.Error(err, "Can'c get namespaces list")
	}

	// wait for client ready
	time.Sleep(3 * time.Second)

	for _, namespace := range namespaces.Items {
		pipelines := new(v1.PipelineList)
		err := c.client.List(context.Background(), pipelines, client.InNamespace(namespace.Name))
		if err != nil {
			c.logger.Error(err, "list pipeline fail...")
			continue
		}

		for _, p := range pipelines.Items {
			if isRegistered(&p) || isExpired(&p) {
				continue
			}

			go c.wait(p.DeepCopy())
		}
	}
}

func (c *CronTaskSet) wait(p *v1.Pipeline) {
	if p.Spec.Strategy == nil {
		return
	}

	if p.Spec.Strategy.CronExpression == "" {
		c.logger.Info("cronExpression is null.... [skip] ")
		return
	}

	if p.Spec.BatchJob == nil {
		c.logger.Info("cron task can't support streamJob.... [skip] ")
		return
	}

	sched, err := cron.ParseStandard(p.Spec.Strategy.CronExpression)
	if err != nil {
		c.logger.Info("unparseable schedule.", "expression", p.Spec.Strategy.CronExpression)
		return
	}

	cr := cron.New()
	cr.Schedule(sched, cron.FuncJob(p.CronTask))
	cr.Start()

	finish := make(chan bool)
	key := fmt.Sprintf("%s/%s", p.Namespace, p.Name)

	registerMut.Lock()
	registerDict[key] = finish
	registerMut.Unlock()

	select {
	case <-c.stopCh:
	case <-finish:
		c.logger.Info("p cron task finished...")
	}

	c.release(p)
	cr.Stop()
}

func (c *CronTaskSet) release(p *v1.Pipeline) {
	key := fmt.Sprintf("%s/%s", p.Namespace, p.Name)

	registerMut.Lock()
	defer registerMut.Unlock()
	if finish, ok := registerDict[key]; ok {
		finish <- true
		delete(registerDict, key)
		c.logger.Info("release pipeline....", "pipeline", p.Name)
	}
}

func isExpired(p *v1.Pipeline) bool {
	if p.Spec.Strategy == nil {
		return false
	}

	if len(p.Spec.Strategy.Expire) == 0 {
		return false
	}

	expireTime, err := time.Parse("2006-01-02 15:04:05", p.Spec.Strategy.Expire)
	if err != nil {
		return false
	}

	if !expireTime.After(time.Now()) {
		logs.Info("pipeline expired %s/%s", p.Namespace, p.Name)
		return true
	}

	return false
}

func isRegistered(p *v1.Pipeline) bool {
	key := fmt.Sprintf("%s/%s", p.Namespace, p.Name)
	if _, ok := registerDict[key]; !ok {
		return false
	}
	return true
}
