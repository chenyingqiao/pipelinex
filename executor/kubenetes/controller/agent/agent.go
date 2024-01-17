package agent

import (
	"github.com/chenyingqiao/pipelinex/executor/kubenetes/controller"
	clientset "github.com/chenyingqiao/pipelinex/executor/kubenetes/generated/clientset/versioned"
	"github.com/chenyingqiao/pipelinex/executor/kubenetes/generated/clientset/versioned/scheme"
	informers "github.com/chenyingqiao/pipelinex/executor/kubenetes/generated/informers/externalversions"
	listers "github.com/chenyingqiao/pipelinex/executor/kubenetes/generated/listers/agentcontroller/v1alpha1"
	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
	kuberInformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	kubeScheme "k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/apimachinery/pkg/util/runtime"
)

const controllerAgentName = "agent-controller"

// 这个是Agent工作负载编排的控制器
// 它的功能：
// - 创建流水线工作负载的Pod, 并保证工作负载的pod创建成功
// - 再时间到了的时候销毁工作负载, 销毁工作负载时，这个工作负载应该是没有任何流水线正在运行的
// - 记录工作负载的状态以及包含了那些流水线
// - 到容器中执行命令，并且获取日志信息
type Controller struct {
	// kubeclientset 是标准的 kubernetes 客户端集
	kubeclientset kubernetes.Interface
	// agentclientset 是我们自己的 api 组的客户端集
	agentclientset clientset.Interface

	agentLister listers.AgentLister
	agentSynced cache.InformerSynced

	// workqueue 是一个速率受限的工作队列.
	// workqueue 是一个速率受限的工作队列. 这用于对要处理的工作进行排队，而不是在发生更改时立即执行它。
	//这意味着我们可以确保一次只处理固定数量的资源，并且可以轻松确保我们永远不会在两个不同的工作人员中同时处理同一项目。
	workqueue workqueue.RateLimitingInterface
	// recorder是事件记录器，用于将事件资源记录到kubernetes api。
	recorder record.EventRecorder
}

func NewController(kubeclientset kubernetes.Interface, agentclientset clientset.Interface, kubernetesInformer kuberInformers.SharedInformerFactory, agentInformers informers.SharedInformerFactory) controller.Interface {
	agentInformer := agentInformers.Agentcontroller().V1alpha1().Agents()
	scheme.AddToScheme(kubeScheme.Scheme)
	glog.V(4).Info("Creating event broadcaster")
	// 创建事件广播器
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	// 实例化controller
	ctr := &Controller{
		kubeclientset:  kubeclientset,
		agentclientset: agentclientset,
		agentLister:    agentInformer.Lister(),
		agentSynced:    agentInformer.Informer().HasSynced,
		workqueue:      workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Agent"),
		recorder:       recorder,
	}
	glog.Info("Setting up event handlers")
	// Set up an event handler for when Agent resources change
	agentInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: ctr.enqueueBar,
		UpdateFunc: func(old, new interface{}) {
			ctr.enqueueBar(new)
		},
	})
	return ctr
}

func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	return nil
}
// enqueue bar 获取一个 bar 资源并将其转换为名称空间/名称 字符串，然后将其放入工作队列中
//此方法不应该传递除 bar 之外的任何类型的资源。
func (c *Controller) enqueueBar(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	c.workqueue.AddRateLimited(key)
}
