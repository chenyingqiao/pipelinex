package agent

import (
	"github.com/chenyingqiao/pipelinex/executor/kubenetes/controller"
	clientset "github.com/chenyingqiao/pipelinex/executor/kubenetes/generated/clientset/versioned"
	informers "github.com/chenyingqiao/pipelinex/executor/kubenetes/generated/informers/externalversions"
	listers "github.com/chenyingqiao/pipelinex/executor/kubenetes/generated/listers/agentcontroller/v1alpha1"
	kuberInformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
)

// Controller is the controller implementation for Bar resources
type Controller struct {
	// kubeclientset 是标准的 kubernetes 客户端集
	kubeclientset kubernetes.Interface
	// agentclientset 是我们自己的 api 组的客户端集
	agentclientset clientset.Interface

	barsLister listers.AgentLister
	barsSynced cache.InformerSynced

	// workqueue 是一个速率受限的工作队列.
	// workqueue 是一个速率受限的工作队列. 这用于对要处理的工作进行排队，而不是在发生更改时立即执行它。
	//这意味着我们可以确保一次只处理固定数量的资源，并且可以轻松确保我们永远不会在两个不同的工作人员中同时处理同一项目。
	workqueue workqueue.RateLimitingInterface
	// recorder是事件记录器，用于将事件资源记录到kubernetes api。
	recorder record.EventRecorder
}

func NewController(kubeclientset kubernetes.Interface, agentclientset clientset.Interface, kubernetesInformer kuberInformers.SharedInformerFactory, agentInformers informers.SharedInformerFactory) controller.Interface {
	return nil
}
