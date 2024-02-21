package kubenetes

import (
	"context"
	"time"

	"github.com/chenyingqiao/pipelinex/executor/kubenetes/controller"
	"github.com/chenyingqiao/pipelinex/executor/kubenetes/controller/agent"
	"github.com/chenyingqiao/pipelinex/executor/kubenetes/generated/clientset/versioned"
	"github.com/chenyingqiao/pipelinex/executor/kubenetes/generated/informers/externalversions"
	"github.com/golang/glog"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	threadsPerController = 1
)

type StartupParam struct {
	Kubeconfig string
	MasterUrl  string
}

func Boot(ctx context.Context, param StartupParam) {
	cfg, err := clientcmd.BuildConfigFromFlags(param.MasterUrl, param.Kubeconfig)
	if err != nil {
		glog.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	// 实例化k8s自带的informer以及自定义的informer
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}
	kubeInformer := informers.NewSharedInformerFactory(kubeClient, time.Second*10)

	agentClient, err := versioned.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("Error building example clientset: %s", err.Error())
	}
	agentInformer := externalversions.NewSharedInformerFactory(agentClient, time.Second*10)

	// 定义控制器列表
	ctors := []controller.Constructor{
		agent.NewController,
	}

	controllers := make([]controller.Interface, 0, len(ctors))
	for _, ctor := range ctors {
		controllers = append(controllers,
			ctor(kubeClient, agentClient, kubeInformer, agentInformer))
	}

	// 启动informer
	stopC := ctx.Done()
	kubeInformer.Start(stopC)
	agentInformer.Start(stopC)

	for _, ctrlr := range controllers {
		go func(ctrlr controller.Interface) {
			if err := ctrlr.Run(threadsPerController, stopC); err != nil {
				glog.Fatalf("Error running controller: %s", err.Error())
			}
		}(ctrlr)
	}
	glog.Flush()
}
