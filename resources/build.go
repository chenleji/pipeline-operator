package resources

import (
	"crypto/rand"
	"encoding/hex"
	"github.com/chenleji/pipeline-operator/api/v1"
	"github.com/knative/build/pkg/apis/build/v1alpha1"
	"io"
	"io/ioutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// labels
	LabelKeyPipeline    = "pipeline"
	LabelKeyPipelineRun = "pipeline-run"
	LabelKeyAppName     = "k8s-app"
	// pipeline
	PipelineInputName  = "input"
	PipelineOutputName = "output-"
	// service
	k8sHeadlessServiceClusterIP = "None"
	// image
	ImageInputEsKey         = "inputESKey"
	ImageInputApiServerKey  = "inputApiServerKey"
	ImageOutputApiServerKey = "outputApiServerKey"
	ImageOutputFtpKey       = "outputFtpKey"
	ImageOutputAs2Key       = "outputAs2Key"
	ImageOutputKafkaKey     = "outputKafkaKey"
	ImageOutputEmailKey     = "outputEmailKey"
	RegistryImagePrefix     = "ljchen/"
	ImageKey                = "image"
)

var ImageDict = map[string]string{
	ImageInputEsKey:         RegistryImagePrefix + "ieyes/pier:0.1",
	ImageInputApiServerKey:  RegistryImagePrefix + "ycloud/ieyes/pier:0.1",
	ImageOutputApiServerKey: RegistryImagePrefix + "ycloud/api-adapter:0.1",
	ImageOutputFtpKey:       RegistryImagePrefix + "ycloud/ftp-adapter:0.1",
	ImageOutputAs2Key:       RegistryImagePrefix + "busybox:latest",
	ImageOutputKafkaKey:     RegistryImagePrefix + "ycloud/kafka-adapter:0.1",
	ImageOutputEmailKey:     RegistryImagePrefix + "ycloud/smtp-adapter:0.1",
}

func MakeDeploysAndServices(pipeline *v1.Pipeline, run *v1.PipelineRun) ([]appsv1.Deployment, []corev1.Service) {
	deployments := make([]appsv1.Deployment, 0)
	services := make([]corev1.Service, 0)

	micSvcs := pipeline.Spec.StreamJob.MicroServices
	for _, svc := range micSvcs {

		// selector labels
		podLabels := UnionMaps(pipeline.GetLabels(), map[string]string{
			LabelKeyPipeline:    pipeline.Name,
			LabelKeyPipelineRun: run.Name,
			LabelKeyAppName:     makeDeployName(pipeline.Name, svc.Name),
		})

		// deployment
		deployment := appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:        makeDeployName(pipeline.Name, svc.Name),
				Namespace:   pipeline.Namespace,
				Labels:      podLabels,
				Annotations: CopyMap(pipeline.GetAnnotations()),
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(run.GetObjectMeta(), run.GroupVersionKind()),
				},
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: svc.Replicas,
				Selector: &metav1.LabelSelector{
					MatchLabels: podLabels,
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: podLabels,
					},
					Spec: svc.PodSpec,
				},
			},
		}
		deployments = append(deployments, deployment)

		// service
		service := corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:        svc.Name,
				Namespace:   pipeline.Namespace,
				Labels:      podLabels,
				Annotations: CopyMap(pipeline.GetAnnotations()),
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(run.GetObjectMeta(), run.GroupVersionKind()),
				},
			},
			Spec: corev1.ServiceSpec{
				Selector:  CopyMap(deployment.ObjectMeta.Labels),
				Type:      corev1.ServiceTypeClusterIP,
				ClusterIP: k8sHeadlessServiceClusterIP,
			},
		}
		services = append(services, service)
	}

	return deployments, services
}

func MakeBuild(p *v1.Pipeline, run *v1.PipelineRun) *v1alpha1.Build {
	var buildSteps = make([]corev1.Container, 0)

	if IsStatusRetry(&run.Status.Status) && len(run.Status.StepsCompleted) != 0 {
		buildSteps = getFailedSteps(p, run)
	} else {
		buildSteps = getBuildSteps(p)
	}

	buildName := makeBuildName(run.Name)

	return &v1alpha1.Build{
		ObjectMeta: metav1.ObjectMeta{
			Name:      buildName,
			Namespace: p.Namespace,
			Labels: UnionMaps(p.GetLabels(), map[string]string{
				LabelKeyPipeline:    p.Name,
				LabelKeyPipelineRun: run.Name,
				LabelKeyAppName:     buildName,
			}),
			Annotations: CopyMap(p.GetAnnotations()),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(run.GetObjectMeta(), run.GroupVersionKind()),
			},
		},
		Spec: v1alpha1.BuildSpec{
			Timeout:            run.Spec.Timeout,
			Steps:              buildSteps,
			Volumes:            p.Spec.BatchJob.Volumes,
			ServiceAccountName: p.Spec.BatchJob.ServiceAccountName,
			NodeSelector:       p.Spec.BatchJob.NodeSelector,
			Affinity:           p.Spec.BatchJob.Affinity,
		},
	}
}

func GetRandomString(l int64) string {
	var randReader = rand.Reader
	b, err := ioutil.ReadAll(io.LimitReader(randReader, l))
	if err != nil {
		panic("gun")
	}
	return hex.EncodeToString(b)
}

func MergePipelineRunStatus(run *v1.PipelineRun, build *v1alpha1.Build) *v1.PipelineRun {

	if build.Status.StartTime != nil {
		run.Status.StartTime = build.Status.StartTime
	}

	if build.Status.CompletionTime != nil {
		run.Status.CompletionTime = build.Status.CompletionTime
	}

	run.Status.StepsCompleted = build.Status.StepsCompleted
	run.Status.StepStatus = build.Status.StepStates

	newConds := build.Status.GetConditions()
	run.Status.SetConditions(newConds)

	return run
}

func getFailedSteps(p *v1.Pipeline, run *v1.PipelineRun) []corev1.Container {
	retrySteps := make([]corev1.Container, 0)

	if IsStatusFailedTimeout(&run.Status.Status) {
		retrySteps = getBuildSteps(p)
	} else {
		lastSuccessStepIdx := 0
		skipStepName := ""

		// find last success step index
		for i, step := range run.Status.StepStatus {
			if step.Terminated != nil {
				if step.Terminated.ExitCode != 0 {
					break
				}
				lastSuccessStepIdx = i
			}
		}

		// get success step name
		if len(run.Status.StepsCompleted) != 0 {
			skipStepName = run.Status.StepsCompleted[lastSuccessStepIdx]
		}

		allBuildSteps := getBuildSteps(p)
		found := false
		for _, step := range allBuildSteps {
			if step.Name == skipStepName[len("build-step-"):] {
				found = true
				continue
			}

			if found {
				retrySteps = append(retrySteps, step)
			}
		}
	}

	return retrySteps
}

func getBuildSteps(p *v1.Pipeline) []corev1.Container {
	var buildSteps = make([]corev1.Container, 0)

	// Input stage
	inputStep := makeInputStep(p.Spec.BatchJob.Input, p.Spec.BatchJob.Resources)
	buildSteps = append(buildSteps, inputStep)

	// Processor stages
	for _, step := range p.Spec.BatchJob.Processors {
		step.Resources = p.Spec.BatchJob.Resources
		buildSteps = append(buildSteps, step)
	}

	// Output stages
	for _, output := range p.Spec.BatchJob.Outputs {
		var outputSteps = makeOutputSteps(output, p.Spec.BatchJob.Resources)
		for _, outputStep := range outputSteps {
			buildSteps = append(buildSteps, outputStep)
		}
	}

	return buildSteps
}

func makeBuildName(parent string) string {

	return parent + "-build-" + GetRandomString(3)
}

func makeDeployName(parent, deployName string) string {
	return parent + "-deploy-" + deployName
}

func makeServiceName(parent, serviceName string) string {
	return parent + "-svc-" + serviceName
}

func makeOutputStepName(name string) string {
	return PipelineOutputName + "-" + name
}

func makeInputStep(input *v1.PipelineBatchJobInput, resource corev1.ResourceRequirements) corev1.Container {
	var step = corev1.Container{
		Name:            PipelineInputName,
		Resources:       resource,
		ImagePullPolicy: corev1.PullAlways,
	}

	if input.ElasticSearch != nil {
		step.Image = ImageDict[ImageInputEsKey]
		for k, v := range input.ElasticSearch {
			step.Args = append(step.Args, "--"+k+"="+v)
		}
	} else if input.ApiServer != nil {
		step.Image = ImageDict[ImageInputApiServerKey]
		for k, v := range input.ApiServer {
			step.Args = append(step.Args, "--"+k+"="+v)
		}
	}

	//// for debug TODO
	//step.Image = ImageDict[ImageInputEsKey]
	//step.Command = []string{"/bin/sh"}
	//step.Args = []string{"-c", "sleep 5"}
	//// end debug

	return step
}

func makeOutputSteps(output *v1.PipelineBatchJobOutput, resource corev1.ResourceRequirements) []corev1.Container {
	steps := make([]corev1.Container, 0)

	// api-server
	if output.Api != nil {
		var step = corev1.Container{
			Name:            makeOutputStepName(output.Name),
			Resources:       resource,
			Image:           ImageDict[ImageOutputApiServerKey],
			ImagePullPolicy: corev1.PullAlways,
		}

		for k, v := range output.Api {
			step.Args = append(step.Args, "--"+k+"="+v)
		}

		steps = append(steps, step)
	}

	// as2
	if output.As2 != nil {
		var step = corev1.Container{
			Name:            makeOutputStepName(output.Name),
			Resources:       resource,
			Image:           ImageDict[ImageOutputAs2Key],
			ImagePullPolicy: corev1.PullAlways,
		}

		for k, v := range output.As2 {
			step.Args = append(step.Args, "--"+k+"="+v)
		}

		steps = append(steps, step)
	}

	// ftp
	if output.Ftp != nil {
		var step = corev1.Container{
			Name:            makeOutputStepName(output.Name),
			Resources:       resource,
			Image:           ImageDict[ImageOutputFtpKey],
			ImagePullPolicy: corev1.PullAlways,
		}

		for k, v := range output.Ftp {
			step.Args = append(step.Args, "--"+k+"="+v)
		}

		steps = append(steps, step)
	}

	// kafka
	if output.Kafka != nil {
		var step = corev1.Container{
			Name:            makeOutputStepName(output.Name),
			Resources:       resource,
			Image:           ImageDict[ImageOutputKafkaKey],
			ImagePullPolicy: corev1.PullAlways,
		}

		for k, v := range output.Kafka {
			step.Args = append(step.Args, "--"+k+"="+v)
		}

		steps = append(steps, step)
	}

	// email
	if output.Email != nil {
		var step = corev1.Container{
			Name:            makeOutputStepName(output.Name),
			Resources:       resource,
			Image:           ImageDict[ImageOutputEmailKey],
			ImagePullPolicy: corev1.PullAlways,
		}

		for k, v := range output.Email {
			step.Args = append(step.Args, "--"+k+"="+v)
		}

		steps = append(steps, step)
	}

	// custom
	if output.Custom != nil {
		customImage := ""
		for k, v := range output.Custom {
			if k == ImageKey {
				customImage = v
				break
			}
		}

		if customImage != "" {
			var step = corev1.Container{
				Name:            makeOutputStepName(output.Name),
				Resources:       resource,
				Image:           customImage,
				ImagePullPolicy: corev1.PullAlways,
			}

			for k, v := range output.Custom {
				if k == ImageKey {
					continue
				}
				step.Args = append(step.Args, "--"+k+"="+v)
			}

			steps = append(steps, step)
		}
	}

	//// for debug TODO
	//step := corev1.Container{
	//	Name:      makeOutputStepName(output.Name),
	//	Resources: resource,
	//	Image:     ImageDict[ImageOutputFtpKey],
	//	//Command:   []string{"/bin/sh"},
	//	Args: []string{"-c", "sleep 5"},
	//}
	//
	//steps = []corev1.Container{}
	//steps = append(steps, step)
	//// end debug

	return steps
}
