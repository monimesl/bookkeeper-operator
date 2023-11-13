/*
 * Copyright 2021 - now, the original author or authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *       https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package v1alpha1

import (
	"k8s.io/api/core/v1"
	"time"
)

type ConditionType string

const (
	ConditionClusterPreparing ConditionType = "Preparing"
	ConditionClusterReady     ConditionType = "Ready"
	ConditionClusterError     ConditionType = "Error"
)

// BookkeeperClusterStatus defines the observed state of BookkeeperCluster
type BookkeeperClusterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// Conditions list all the applied conditions
	Conditions []ClusterCondition `json:"conditions,omitempty"`

	// Replicas is the number of desired bookkeeper nodes in the cluster
	// +optional
	Replicas int32 `json:"replicas"`

	// CurrentReplicas is the number of current bookkeeper nodes in the cluster
	// +optional
	CurrentReplicas int32 `json:"currentReplicas"`

	// ReadyReplicas is the number of ready bookkeeper nodes in the cluster
	// +optional
	ReadyReplicas int32 `json:"readyReplicas"`

	// Membership describe the status of members within the cluster
	// +optional
	Membership Membership `json:"members"`
	// Metadata defines the metadata status of the cluster
	// +optional
	Metadata Metadata `json:"metadata,omitempty"`
}

// Metadata defines the metadata status of the cluster
type Metadata struct {
	Size                  int32   `json:"size,omitempty"`
	ServiceMonitorVersion *string `json:"serviceMonitorVersion,omitempty"`
}

// Membership is the status of the members within the cluster
type Membership struct {
	// +optional
	// +nullable
	Ready []string `json:"ready"`
	// +optional
	// +nullable
	Unready []string `json:"unready"`
}

// ClusterCondition describes the current cluster condition.
// This is a compliance to kubernetes Object API convention
type ClusterCondition struct {
	// Type is the type of cluster condition.
	// +optional
	Type ConditionType `json:"type"`

	// Status of the condition, one of True, False, Unknown.
	// +optional
	Status v1.ConditionStatus `json:"status"`

	// Reason describes why did the last transition occurred.
	// +optional
	Reason string `json:"reason,omitempty"`

	// Message is detailed description of the transition.
	// +optional
	Message string `json:"message,omitempty"`

	// LastUpdateTime the last time the condition was updated.
	// +optional
	LastUpdateTime string `json:"lastUpdateTime,omitempty"`

	// LastTransitionTime the last time a transition was made.
	// +optional
	LastTransitionTime string `json:"lastTransitionTime,omitempty"`
}

func (in *BookkeeperClusterStatus) setDefault() (changed bool) {
	clusterConditions := []ConditionType{
		ConditionClusterPreparing,
		ConditionClusterReady,
		ConditionClusterError,
	}
	if in.Conditions == nil {
		changed = true
		for _, typ := range clusterConditions {
			in.setCondition(typ, v1.ConditionFalse, "", "")
		}
	}
	return
}

func (in *BookkeeperClusterStatus) SetPodsReadyConditionTrue() {
	in.setCondition(ConditionClusterReady, v1.ConditionTrue, "", "")
}

func (in *BookkeeperClusterStatus) SetPodsReadyConditionFalse() {
	in.setCondition(ConditionClusterReady, v1.ConditionFalse, "", "")
}

func (in *BookkeeperClusterStatus) SetPreparingCondition() {
	in.setCondition(ConditionClusterPreparing, v1.ConditionTrue, "deploying the pods", "")
}

func (in *BookkeeperClusterStatus) GetCondition(typ ConditionType) (int, *ClusterCondition) {
	for i, condition := range in.Conditions {
		if condition.Type == typ {
			return i, &condition
		}
	}
	return -1, nil
}

func (in *BookkeeperClusterStatus) setCondition(condType ConditionType, status v1.ConditionStatus, reason, message string) {
	newCond := &ClusterCondition{
		Type:               condType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastUpdateTime:     "",
		LastTransitionTime: "",
	}
	now := time.Now().Format(time.RFC3339)
	position, old := in.GetCondition(newCond.Type)
	if old != nil {
		if old.Status != newCond.Status {
			old.Status = newCond.Status
			old.LastTransitionTime = now
			old.LastUpdateTime = now
		}

		if old.Reason != newCond.Reason || old.Message != newCond.Message {
			old.Reason = newCond.Reason
			old.Message = newCond.Message
			old.LastUpdateTime = now
		}
		in.Conditions[position] = *old
		return
	}
	in.Conditions = append(in.Conditions, *newCond)
}
