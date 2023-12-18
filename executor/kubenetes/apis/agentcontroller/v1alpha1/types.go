package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Agent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AgentSpec   `json:"spec"`
	Status AgentStatus `json:"status"`
}

// Agent 的规格参数结构体
type AgentSpec struct {
	LiveTime int `json:"live_time"`
}

// Agent的状态描述结构体
type AgentStatus struct {
	Pipelines []PipelineStatus `json:"pipelines"`
}

type PipelineStatus struct {
	ID     string `json:"id"`
	Pod    string `json:"pod"`
	Status string `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type AgentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Agent `json:"items"`
}
