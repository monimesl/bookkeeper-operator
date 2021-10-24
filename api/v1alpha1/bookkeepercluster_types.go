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
	"fmt"
	"github.com/monimesl/operator-helper/basetype"
	"github.com/monimesl/operator-helper/config"
	"github.com/monimesl/operator-helper/k8s"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

//+kubebuilder:object:root=true

// BookkeeperClusterList contains a list of BookkeeperCluster
type BookkeeperClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BookkeeperCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BookkeeperCluster{}, &BookkeeperClusterList{})
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// BookkeeperCluster is the Schema for the bookkeeperclusters API
type BookkeeperCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BookkeeperClusterSpec   `json:"spec,omitempty"`
	Status BookkeeperClusterStatus `json:"status,omitempty"`
}

// SetSpecDefaults set the defaults for the cluster spec and returns true otherwise false
func (in *BookkeeperCluster) SetSpecDefaults() (changed bool) {
	return in.Spec.setDefaults()
}

// SetStatusDefaults set the defaults for the cluster status and returns true otherwise false
func (in *BookkeeperCluster) SetStatusDefaults() (changed bool) {
	return in.Status.setDefault()
}

func (in *BookkeeperCluster) CreateLabels(addPodLabels bool, more map[string]string) map[string]string {
	return in.Spec.createLabels(in.Name, addPodLabels, more)
}

func (in *BookkeeperCluster) generateName() string {
	return in.GetName()
}

// ConfigMapName defines the name of the configmap object
func (in *BookkeeperCluster) ConfigMapName() string {
	return in.generateName()
}

// StatefulSetName defines the name of the statefulset object
func (in *BookkeeperCluster) StatefulSetName() string {
	return in.generateName()
}

// ClientServiceName defines the name of the client service object
func (in *BookkeeperCluster) ClientServiceName() string {
	return in.generateName()
}

// HeadlessServiceName defines the name of the headless service object
func (in *BookkeeperCluster) HeadlessServiceName() string {
	return fmt.Sprintf("%s-headless", in.ClientServiceName())
}

// ClientServiceFQDN defines the FQDN of the client service object
func (in *BookkeeperCluster) ClientServiceFQDN() string {
	return fmt.Sprintf("%s.%s.svc.%s", in.ClientServiceName(), in.Namespace, in.Spec.ClusterDomain)
}

// ZkRootPath the zk root of this bookkeeper cluster
func (in *BookkeeperCluster) ZkRootPath() string {
	return fmt.Sprintf("/bookkeeper/%s", in.Name)
}

// ZkLedgersRootPath the zkLedgersRootPath of this bookkeeper cluster
func (in *BookkeeperCluster) ZkLedgersRootPath() string {
	return fmt.Sprintf("%s/ledgers", in.ZkRootPath())
}

// ShouldDeleteStorage returns whether the PV should should be deleted or not
func (in *BookkeeperCluster) ShouldDeleteStorage() bool {
	return in.Spec.Persistence.ReclaimPolicy == VolumeReclaimPolicyDelete
}

// WaitClusterTermination wait for all the bookkeeper pods in cluster to terminated
func (in *BookkeeperCluster) WaitClusterTermination(kubeClient client.Client) (err error) {
	config.RequireRootLogger().Info(
		"Waiting for the cluster to terminate",
		"cluster", in.GetName())
	labels := in.CreateLabels(true, nil)
	return k8s.WaitForPodsToTerminate(kubeClient, in.Namespace, labels)
}

// Image the bookkeeper docker image for the cluster
func (in *BookkeeperCluster) Image() basetype.Image {
	return basetype.Image{
		Repository: imageRepository,
		PullPolicy: in.Spec.ImagePullPolicy,
		Tag:        in.Spec.BookkeeperVersion,
	}
}
