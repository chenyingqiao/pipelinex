package controller

import (
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"

	clientset "github.com/chenyingqiao/pipelinex/executor/kubenetes/generated/clientset/versioned"
	informers "github.com/chenyingqiao/pipelinex/executor/kubenetes/generated/informers/externalversions"
)

type Interface interface {
	Run(threadiness int, stopCh <-chan struct{}) error
}

type Constructor func(kubernetes.Interface, clientset.Interface, kubeinformers.SharedInformerFactory, informers.SharedInformerFactory) Interface
