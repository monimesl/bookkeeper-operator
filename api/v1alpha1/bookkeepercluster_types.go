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
	"github.com/monimesl/operator-helper/k8s"
	"github.com/monimesl/operator-helper/k8s/pod"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
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

// BookkeeperClusterSpec defines the desired state of BookkeeperCluster
type BookkeeperClusterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Image defines the container image to use.
	// +optional
	Image basetype.Image `json:"image,omitempty"`
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	Size int32 `json:"size,omitempty"`
	// MaxUnavailableNodes defines the maximum number of nodes that
	// can be unavailable as per kubernetes PodDisruptionBudget
	// Default is 1.
	// +optional
	MaxUnavailableNodes int32 `json:"maxUnavailableNodes"`
	// ZookeeperUrl specifies the hostname/IP address and port in the format "hostname:port".
	// +kubebuilder:validation:Required
	ZookeeperUrl string       `json:"zookeeperUri"`
	Directories  *Directories `json:"directories,omitempty"`
	Ports        *Ports       `json:"ports,omitempty"`
	// EnableAutoRecovery indicates whether or not BookKeeper auto recovery is enabled.
	// Defaults to true.
	// +optional
	EnableAutoRecovery *bool `json:"enableAutoRecovery"`
	// JVMOptions defines the JVM options for bookkeeper; this is useful for performance tuning.
	// If unspecified, a reasonable defaults will be set
	// +optional
	JVMOptions *JVMOptions `json:"jvmOptions"`
	// Configs defines the Bookkeeper configurations to override the bk_server.conf
	// https://github.com/apache/bookkeeper/tree/master/docker#configuration
	// +optional
	Configs map[string]string `json:"configs"`
	// PodConfig defines common configuration for the bookkeeper pods
	PodConfig basetype.PodConfig `json:"pod,omitempty"`
	// Probes defines the probing settings for the bookkeeper containers
	Probes *pod.Probes `json:"probes,omitempty"`
	// Env defines environment variables for the bookkeeper statefulset pods
	Env []v1.EnvVar `json:"env,omitempty"`

	Persistence *Persistence `json:"persistence,omitempty"`

	// Labels defines the labels to attach to the bookkeeper deployment
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations defines the annotations to attach to the bookkeeper deployment
	Annotations map[string]string `json:"annotations,omitempty"`

	// ClusterDomain defines the cluster domain for the cluster
	// It defaults to cluster.local
	ClusterDomain string `json:"clusterDomain,omitempty"`
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

func (in *BookkeeperCluster) nameHasBkIndicator() bool {
	return strings.Contains(in.Name, "bk") || strings.Contains(in.Name, "bookkeeper")
}

// ConfigMapName defines the name of the configmap object
func (in *BookkeeperCluster) ConfigMapName() string {
	if in.nameHasBkIndicator() {
		return in.Name
	}
	return fmt.Sprintf("%s-bk", in.Name)
}

// StatefulSetName defines the name of the statefulset object
func (in *BookkeeperCluster) StatefulSetName() string {
	if in.nameHasBkIndicator() {
		return in.Name
	}
	return fmt.Sprintf("%s-zk", in.GetName())
}

// ClientServiceName defines the name of the client service object
func (in *BookkeeperCluster) ClientServiceName() string {
	if in.nameHasBkIndicator() {
		return fmt.Sprintf("%s", in.GetName())
	}
	return fmt.Sprintf("%s-zk", in.GetName())
}

// HeadlessServiceName defines the name of the headless service object
func (in *BookkeeperCluster) HeadlessServiceName() string {
	return fmt.Sprintf("%s-headless", in.ClientServiceName())
}

// ClientServiceFQDN defines the FQDN of the client service object
func (in *BookkeeperCluster) ClientServiceFQDN() string {
	return fmt.Sprintf("%s.%s.svc.%s", in.ClientServiceName(), in.Namespace, in.Spec.ClusterDomain)
}

// HeadlessServiceFQDN defines the FQDN of the headless service object
func (in *BookkeeperCluster) HeadlessServiceFQDN() string {
	return fmt.Sprintf("%s.%s.svc.%s", in.HeadlessServiceName(), in.Namespace, in.Spec.ClusterDomain)
}

// WaitClusterTermination wait for all the bookkeeper pods in cluster to terminated
func (in *BookkeeperCluster) WaitClusterTermination(kubeClient client.Client) (err error) {
	labels := in.CreateLabels(true, nil)
	return k8s.WaitForPodsToTerminate(kubeClient, in.Namespace, labels)
}
