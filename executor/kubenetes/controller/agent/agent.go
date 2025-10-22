package agent

import (
	"fmt"
	"time"

	"github.com/chenyingqiao/pipelinex/executor/kubenetes/controller"
	clientset "github.com/chenyingqiao/pipelinex/executor/kubenetes/generated/clientset/versioned"
	"github.com/chenyingqiao/pipelinex/executor/kubenetes/generated/clientset/versioned/scheme"
	informers "github.com/chenyingqiao/pipelinex/executor/kubenetes/generated/informers/externalversions"
	listers "github.com/chenyingqiao/pipelinex/executor/kubenetes/generated/listers/agentcontroller/v1alpha1"
	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	kuberInformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	kubeScheme "k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
)

const controllerAgentName = "agent-controller"

const (
	// SuccessSynced is used as part of the Event 'reason' when a Bar is synced
	SuccessSynced = "Synced"
	// ErrResourceExists is used as part of the Event 'reason' when a Bar fails
	// to sync due to a Deployment of the same name already existing.
	ErrResourceExists = "ErrResourceExists"

	// MessageResourceSynced is the message used for an Event fired when a Bar
	// is synced successfully
	MessageResourceSynced = "Bar synced successfully"
)

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
	// create event broadcaster
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	// new controller
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

// 执行控制器功能
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	glog.Info("开始运行AgentController")
	glog.Info("等待informer同步信息")
	if ok := cache.WaitForCacheSync(stopCh, c.agentSynced); !ok {
		return fmt.Errorf("等待informer同步失败")
	}

	// 开始worker
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}
	glog.Info("开始工作业务")
	<-stopCh
	glog.Info("结束工作业务")

	return nil
}

func (c *Controller) runWorker() {
	for {
		// 获取工作队列里面的对象
		obj, shutdown := c.workqueue.Get()
		if shutdown {
			return
		}
		defer c.workqueue.Done(obj)
		// 判断是否是字符串
		if _, ok := obj.(string); !ok {
			c.workqueue.Forget(obj)
			runtime.HandleError(fmt.Errorf("工作队列中预期的字符串但得到了 %#v", obj))
			return
		}

		key := obj.(string)
		if err := c.syncHandler(key); err != nil {
			runtime.HandleError(fmt.Errorf("同步数据失败 %#v %s", obj, err.Error()))
			return
		}
		c.workqueue.Forget(obj)
		glog.Infof("Successfully synced '%s'", key)
		return
	}
}

// processing 处理对应的数据
func (c *Controller) syncHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	agent, err := c.agentLister.Agents(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			runtime.HandleError(fmt.Errorf("bar '%s' in work queue no longer exists", key))
			return nil
		}
		return err
	}

	// 处理agent对应的业务逻辑
	// 1. 如果agent的参数变化了需要同步的对pod进行操作
	fmt.Println(agent.Name)
	c.recorder.Event(agent, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
	return nil
}

// enqueue bar 获取一个 bar 资源并将其转换为名称空间/名称 字符串，然后将其放入工作队列中
// 此方法不应该传递除 bar 之外的任何类型的资源。
func (c *Controller) enqueueBar(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	c.workqueue.AddRateLimited(key)
}
