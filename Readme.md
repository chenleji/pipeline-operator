# pipeline-operator

## 功能介绍

该项目是数据交换平台的底层框架，数据交换平台支持两种类型的数据任务：1. 定时或即时批量执行自定义steps； 2. 运行daemon类型的任务，处理流数据。
CRD定义了pipeline，用来统一抽象这两种任务，其中针对第一种，batchjob类型与之适配；针对第二种，streamjob类型与之适配。

batchjob可能会有定期执行的要求，因此，pipeline中有定时器来触发这类pipeline；每次执行就会生成一个pipelinerun。pipelinerun中记录了每次执行的时间，各个步骤的执行情况以及执行状态等。
如果某个阶段执行失败，响应的原因也会被记录到pipelinerun中。另外，pipelinerun还可以设置最大重试次数，在未达到最大重试次数时，会重复执行先前未执行完成的阶段。

> [依赖]
>   该项目依赖于knative building，需要预先部署该项目。具体操作流程见knative官网，直接下载buiding的yaml文件部署即可。 

## 目录结构

pipeline-operator中定义了两个CRD，分别是pipeline和pipelinerun。其中pipeline用于batchjob和streamjob两种job的定义，每次需要执行该pipeline就手动或者自动创建一个pipelinerun。整个运行状态都保存在pipelinerun的status中。
 
由于该项目使用kubebuilder来生成主体框架，该项目无法自动生成client代码，因此在api目录下有两个目录。
 
- `v1`
  kubebuilder使用
  
- `controller`
  按照k8s codegenerator的规则来定义文件结构，内部的数据结构需要与`v1`中的保持一致，然后按照以下方法来自动生成client端代码。
 
# 操作流程
 
## 生成client代码
 
```bash
  cd $GOPATH
  $GOPATH/src/k8s.io/code-generator/generate-groups.sh all github.com/chenleji/pipeline-operator/generated github.com/chenleji/pipeline-operator/api controller:v1
  
```

## 部署

需要执行命令的本机能够已经配置好了kubeconfig，部署的目标机即为对应的k8s集群。

```bash
make deploy
```

# 实例

## pipeline

以下yaml文件定义了一条batchJob类型的pipeline，用于从input（elasticsearch）中采集数据，然后分别执行多个outputs，将数据分别输送到ftp, api-server, kafka, email, custom这些适配器上。
用户通过访问这些适配器之一来获取到最终数据。在input和output中定义的每一个类型都有对应的容器镜像，实现了抽象逻辑。当然，平台也支持custom类型，需要在参数中指定镜像名称和参数。

```yaml
apiVersion: controller.ljchen.net/v1
kind: Pipeline
metadata:
  name: pipeline-test-1
  finalizers:
    - finalizer.ljchen.net
spec:
  batchJob:
    input:
      elasticSearch:
        index: xmtc_stock_test
        topic: xmtc_stock_test3
        es-address: http://10.0.66.60:9200
    outputs:
      - name: "output-ftp"
        ftp:
          bootstrap: kfk-test-elk01.xxx.cn:9092
          filename: test
          filetype: zip
          topic: topic-ddx-xmtc
      - name: "output-api"
        api:
          bootstrap: kfk-test-elk01.xxx.cn:9092
          filename: test
          filetype: zip
          topic: topic-ddx-xmtc
      - name: "output-kafka"
        kafka:
          bootstrap: kfk-test-elk01.xxx.cn:9092
          tobootstrap: kfk-test-elk01.xxx.cn:9092
          totopic: lys-test
          topic: topic-ddx-xmtc
      - name: "output-smtp"
        email:
          smtpserver: smtp.163.com
          smtpuser: xxxxxxxxx@163.com
          smtppassword: xxxxxx
      - name: "custom-server"
        custom:
          image: registry.xx.cn/ddx/adapter4xmtc:0.1
          target: stock
          topic: xmtc_stock_test
  strategy:
    cronExpression: "0/5 * * * *"  # 触发执行周期
    expire: "2019-07-15 19:55:00"  # 终结自动触发执行日期
```

下面定义了streamJob类型的一条pipeline, 基于其中定义的microServices，operator会创建对应的deployment和services，并持续执行。此时，cronExpression不生效。
```yaml
apiVersion: controller.ljchen.net/v1
kind: Pipeline
metadata:
  name: pipeline-test-2
  finalizers:
    - finalizer.ljchen.net
spec:
  streamJob:
    microServices:
      - name: micro-service-1
        replicas: 1
        spec:
          containers:
            - name: nginx-1
              image: nginx
      - name: micro-service-2
        replicas: 1
        spec:
          containers:
            - name: nginx-2
              image: nginx
  strategy:
    expire: "2019-07-15 15:22:00"
    cronExpression: "0/1 * * * *"
```

## pipelinerun

定义了对指定pipeline的一次即时的运行时，该类型条目可能是由用户手动创建，也可能是由pipeline基于cronExpression自动生成。

```yaml
apiVersion: controller.ljchen.net/v1
kind: PipelineRun
metadata:
  name: pipelinerun-1
  finalizers:
    - finalizer.ljchen.net
spec:
  refPipeline: pipeline-test-1  # 对应的任务描述，也就是pipeline名称
  maxRetryCount: 2              # 最大重试次数
  timeout: "30s"                # 在该时间内没有执行成功就认为失败
```

 
