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
	"github.com/monimesl/bookkeeper-operator/internal"
	"github.com/monimesl/operator-helper/basetype"
	"github.com/monimesl/operator-helper/k8s"
	"github.com/monimesl/operator-helper/k8s/pod"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"strings"
)

const (
	minimumClusterSize = 3
	defaultJournalDir  = "/bk/data/journal"
	defaultLedgerDirs  = "/bk/data/ledger"
	defaultIndexDirs   = "/bk/data/index"
)

const (
	imageRepository = "monime/bookkeeper"
	defaultImageTag = "latest"
)

const (
	defaultBookiePort  = 3181
	defaultAdminPort   = 8080
	defaultMetricsPort = 8000
)

const (
	defaultStorageVolumeSize = "10Gi"
	defaultClusterDomain     = "cluster.local"
)

const (
	// VolumeReclaimPolicyDelete deletes the volume after the cluster is deleted
	VolumeReclaimPolicyDelete = "Delete"
	// VolumeReclaimPolicyRetain retains the volume after the cluster is deleted
	VolumeReclaimPolicyRetain = "Retain"
)

const (
	AdminPortName          = "http-admin"
	ClientPortName         = "tcp-client"
	ServiceMetricsPortName = "http-metrics"
)

var (
	defaultTerminationGracePeriod int64 = 600
	defaultClusterSize                  = int32(minimumClusterSize)
)

// BookkeeperClusterSpec defines the desired state of BookkeeperCluster
type BookkeeperClusterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// BookkeeperVersion defines the version of bookkeeper to use
	// +optional
	BookkeeperVersion string `json:"bookkeeperVersion,omitempty"`
	// ImagePullPolicy describes a policy for if/when to pull the image
	// +optional
	ImagePullPolicy v1.PullPolicy `json:"imagePullPolicy,omitempty"`
	// +kubebuilder:validation:Minimum=0
	Size *int32 `json:"size,omitempty"`
	// MaxUnavailableNodes defines the maximum number of nodes that
	// can be unavailable as per kubernetes PodDisruptionBudget
	// Default is 1.
	// +optional
	MaxUnavailableNodes int32 `json:"maxUnavailableNodes"`
	// ZkServers specifies the hostname/IP address and port in the format "hostname:port".
	// +kubebuilder:validation:Required
	ZkServers   string       `json:"zkServers"`
	Directories *Directories `json:"directories,omitempty"`
	Ports       *Ports       `json:"ports,omitempty"`
	// EnableAutoRecovery indicates whether or not BookKeeper auto recovery is enabled.
	// Defaults to true.
	// +optional
	EnableAutoRecovery *bool `json:"enableAutoRecovery"`
	// JVMOptions defines the JVM options for bookkeeper; this is useful for performance tuning.
	// If unspecified, a reasonable defaults will be set
	// +optional
	JVMOptions JVMOptions `json:"jvmOptions"`
	// BkConfig defines the Bookkeeper configurations to override the bk_server.conf
	// https://github.com/apache/bookkeeper/tree/master/docker#configuration
	// +optional
	BkConfig map[string]string `json:"bkConf"`
	// PodConfig defines common configuration for the bookkeeper pods
	// +optional
	PodConfig basetype.PodConfig `json:"podConfig,omitempty"`
	// ProbeConfig defines the probing settings for the bookkeeper containers
	// +optional
	ProbeConfig *pod.Probes `json:"probeConfig,omitempty"`
	// MonitoringConfig
	// +optional
	MonitoringConfig MonitoringConfig `json:"monitoringConfig,omitempty"`
	// Env defines environment variables for the bookkeeper statefulset pods
	Env []v1.EnvVar `json:"env,omitempty"`
	// Persistence configures your node storage
	// +optional
	Persistence *Persistence `json:"persistence,omitempty"`

	// Labels defines the labels to attach to the bookkeeper deployment
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations defines the annotations to attach to the bookkeeper statefulset and services
	Annotations map[string]string `json:"annotations,omitempty"`

	// ClusterDomain defines the cluster domain for the cluster
	// It defaults to cluster.local
	ClusterDomain string `json:"clusterDomain,omitempty"`
}

type MonitoringConfig struct {
	// Enabled defines whether this monitoring is enabled or not.
	Enabled bool `json:"enabled,omitempty"`
}

type Ports struct {
	// +kubebuilder:validation:Minimum=1
	Bookie int32 `json:"bookie,omitempty"`
	// +kubebuilder:validation:Minimum=1
	Admin int32 `json:"admin,omitempty"`
	// +kubebuilder:validation:Minimum=1
	Metrics int32 `json:"metrics,omitempty"`
}

func (in *Ports) setDefaults() (changed bool) {
	if in.Bookie == 0 {
		changed = true
		in.Bookie = defaultBookiePort
	}
	if in.Admin == 0 {
		changed = true
		in.Admin = defaultAdminPort
	}
	if in.Metrics == 0 {
		changed = true
		in.Metrics = defaultMetricsPort
	}
	return
}

type Directories struct {
	IndexDirs  string `json:"indexDirs,omitempty"`
	JournalDir string `json:"journalDir,omitempty"`
	LedgerDirs string `json:"ledgerDirs,omitempty"`
}

type JVMOptions struct {
	// Memory defines memory options
	// +optional
	Memory []string `json:"memory"`
	// Gc defines garbage collection options
	// +optional
	Gc []string `json:"gc"`
	// GcLogging defines garbage collection logging options
	// +optional
	GcLogging []string `json:"gcLogging"`
	// Extra defines extra options
	// +optional
	Extra []string `json:"extra"`
}

func (in *JVMOptions) setDefaults() (changed bool) {
	if in.Memory == nil {
		changed = true
		in.Memory = []string{
			"-Xms128m", "-Xmx256m", "-XX:MaxDirectMemorySize=256m",
		}
	}
	if in.Gc == nil {
		changed = true
		in.Gc = strings.Split(
			"-XX:+UseG1GC -XX:MaxGCPauseMillis=10 -XX:+ParallelRefProcEnabled "+
				"-XX:+UnlockExperimentalVMOptions -XX:+DoEscapeAnalysis -verbosegc "+
				"-XX:ParallelGCThreads=4 -XX:ConcGCThreads=4 -XX:G1NewSizePercent=50 -XX:+DisableExplicitGC "+
				"-XX:-ResizePLAB -XX:+ExitOnOutOfMemoryError -XX:+PerfDisableSharedMem -Xlog:gc* ",
			" ")
	}
	if in.GcLogging == nil {
		changed = true
		in.GcLogging = []string{}
	}
	if in.Extra == nil {
		changed = true
		in.Extra = strings.Split(
			"-Dio.netty.leakDetectionLevel=disabled "+
				"-Dio.netty.recycler.maxCapacity.default=1000 "+
				"-Dio.netty.recycler.linkCapacity=1024", " ")
	}
	return
}

// VolumeReclaimPolicy defines the possible volume reclaim policy: Delete or Retain
type VolumeReclaimPolicy string

// Persistence defines cluster node persistence volume is configured
type Persistence struct {
	// ReclaimPolicy decides the fate of the PVCs after the cluster is deleted.
	// If it's set to Delete and the bookkeeper cluster is deleted, the corresponding
	// PVCs will be deleted. The default value is Retain.
	// +kubebuilder:validation:Enum="Delete";"Retain"
	ReclaimPolicy VolumeReclaimPolicy `json:"reclaimPolicy,omitempty"`
	// JournalVolumeClaimSpec describes the PVC for the bookkeeper journal
	JournalVolumeClaimSpec *v1.PersistentVolumeClaimSpec `json:"journal,omitempty"`
	// LedgerVolumeClaimSpec describes the PVC for the bookkeeper ledgers
	LedgerVolumeClaimSpec *v1.PersistentVolumeClaimSpec `json:"ledger,omitempty"`
	// IndexVolumeClaimSpec describes the PVC for the bookkeeper index
	IndexVolumeClaimSpec *v1.PersistentVolumeClaimSpec `json:"index,omitempty"`
}

func (in *Persistence) setDefault() (changed bool) {
	if in.ReclaimPolicy != VolumeReclaimPolicyDelete && in.ReclaimPolicy != VolumeReclaimPolicyRetain {
		in.ReclaimPolicy = VolumeReclaimPolicyDelete
		changed = true
	}
	if in.JournalVolumeClaimSpec == nil {
		changed = true
		in.JournalVolumeClaimSpec = createVolumeClaimSpec()
	}
	if in.LedgerVolumeClaimSpec == nil {
		changed = true
		in.LedgerVolumeClaimSpec = createVolumeClaimSpec()
	}
	if in.IndexVolumeClaimSpec == nil {
		changed = true
		in.IndexVolumeClaimSpec = createVolumeClaimSpec()
	}
	return
}

func createVolumeClaimSpec() *v1.PersistentVolumeClaimSpec {
	return &v1.PersistentVolumeClaimSpec{
		AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
		Resources: v1.ResourceRequirements{
			Requests: v1.ResourceList{
				v1.ResourceStorage: resource.MustParse(defaultStorageVolumeSize),
			},
		},
	}
}

func (in *BookkeeperClusterSpec) setDefaults() (changed bool) {
	if in.BookkeeperVersion == "" {
		changed = true
		in.BookkeeperVersion = defaultImageTag
	}
	if in.ImagePullPolicy == "" {
		changed = true
		in.ImagePullPolicy = v1.PullIfNotPresent
	}
	if in.Size == nil {
		changed = true
		in.Size = &defaultClusterSize
	}
	if in.MaxUnavailableNodes == 0 {
		changed = true
		in.MaxUnavailableNodes = 1
	}
	if in.ClusterDomain == "" {
		changed = true
		in.ClusterDomain = defaultClusterDomain
	}
	if in.Persistence == nil {
		changed = true
		in.Persistence = &Persistence{}
	}
	if in.Persistence == nil {
		in.Persistence = &Persistence{}
		in.Persistence.setDefault()
		changed = true
	} else if in.Persistence.setDefault() {
		changed = true
	}
	if in.EnableAutoRecovery == nil {
		changed = true
		value := true
		in.EnableAutoRecovery = &value
	}
	if in.BkConfig == nil {
		in.BkConfig = map[string]string{}
	}
	if in.Directories == nil {
		changed = true
		in.Directories = &Directories{
			IndexDirs:  defaultIndexDirs,
			JournalDir: defaultJournalDir,
			LedgerDirs: defaultLedgerDirs,
		}
	}
	if in.Directories.IndexDirs == "" {
		changed = true
		in.Directories.IndexDirs = defaultIndexDirs
	}
	if in.Directories.JournalDir == "" {
		changed = true
		in.Directories.JournalDir = defaultJournalDir
	}
	if in.Directories.LedgerDirs == "" {
		changed = true
		in.Directories.LedgerDirs = defaultLedgerDirs
	}
	if in.Ports == nil {
		in.Ports = &Ports{}
		in.Ports.setDefaults()
		changed = true
	} else if in.Ports.setDefaults() {
		changed = true
	}
	if in.ProbeConfig == nil {
		changed = true
		in.ProbeConfig = &pod.Probes{}
		in.ProbeConfig.SetDefault()
	} else if in.ProbeConfig.SetDefault() {
		changed = true
	}
	if in.JVMOptions.setDefaults() {
		changed = true
	}
	if in.PodConfig.TerminationGracePeriodSeconds == nil {
		changed = true
		in.PodConfig.TerminationGracePeriodSeconds = &defaultTerminationGracePeriod
	}
	return
}

func (in *BookkeeperClusterSpec) createLabels(clusterName string, addPodLabels bool, more map[string]string) map[string]string {
	labels := in.Labels
	if labels == nil {
		labels = map[string]string{}
	}
	if addPodLabels {
		for k, v := range in.PodConfig.Labels {
			labels[k] = v
		}
	}
	for k, v := range more {
		labels[k] = v
	}
	labels["app"] = "bookkeeper"
	labels["version"] = in.BookkeeperVersion
	labels[k8s.LabelAppName] = "bookkeeper"
	labels[k8s.LabelAppInstance] = clusterName
	labels[k8s.LabelAppVersion] = in.BookkeeperVersion
	labels[k8s.LabelAppManagedBy] = internal.OperatorName
	return labels
}
