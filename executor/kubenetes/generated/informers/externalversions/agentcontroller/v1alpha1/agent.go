/*
Copyright The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	time "time"

	agentcontrollerv1alpha1 "github.com/chenyingqiao/pipelinex/executor/kubenetes/apis/agentcontroller/v1alpha1"
	versioned "github.com/chenyingqiao/pipelinex/executor/kubenetes/generated/clientset/versioned"
	internalinterfaces "github.com/chenyingqiao/pipelinex/executor/kubenetes/generated/informers/externalversions/internalinterfaces"
	v1alpha1 "github.com/chenyingqiao/pipelinex/executor/kubenetes/generated/listers/agentcontroller/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// AgentInformer provides access to a shared informer and lister for
// Agents.
type AgentInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.AgentLister
}

type agentInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewAgentInformer constructs a new informer for Agent type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewAgentInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredAgentInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredAgentInformer constructs a new informer for Agent type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredAgentInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.AgentcontrollerV1alpha1().Agents(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.AgentcontrollerV1alpha1().Agents(namespace).Watch(context.TODO(), options)
			},
		},
		&agentcontrollerv1alpha1.Agent{},
		resyncPeriod,
		indexers,
	)
}

func (f *agentInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredAgentInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *agentInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&agentcontrollerv1alpha1.Agent{}, f.defaultInformer)
}

func (f *agentInformer) Lister() v1alpha1.AgentLister {
	return v1alpha1.NewAgentLister(f.Informer().GetIndexer())
}
