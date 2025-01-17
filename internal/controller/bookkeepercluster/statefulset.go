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

package bookkeepercluster

import (
	"context"
	"fmt"
	"github.com/monimesl/bookkeeper-operator/api/v1alpha1"
	"github.com/monimesl/operator-helper/k8s"
	"github.com/monimesl/operator-helper/k8s/pod"
	"github.com/monimesl/operator-helper/k8s/pvc"
	"github.com/monimesl/operator-helper/oputil"
	"github.com/monimesl/operator-helper/reconciler"
	v1 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"strconv"
	"strings"
)

const (
	probeInitialDelaySeconds = 120
	probeFailureThreshold    = 15
	bookieComponent          = "bookie"
)

// ReconcileStatefulSet reconcile the statefulset of the specified cluster
func ReconcileStatefulSet(ctx reconciler.Context, cluster *v1alpha1.BookkeeperCluster) error {
	sts := &v1.StatefulSet{}
	return ctx.GetResource(types.NamespacedName{
		Name:      cluster.StatefulSetName(),
		Namespace: cluster.Namespace,
	}, sts,
		// Found
		func() error {
			if shouldUpdateStatefulSet(ctx, cluster, sts) {
				if err := updateStatefulset(ctx, sts, cluster); err != nil {
					return err
				}
				if err := updateStatefulsetPVCs(ctx, sts, cluster); err != nil {
					return err
				}
			}
			return nil
		},
		// Not Found
		func() error {
			sts = createStatefulSet(cluster)
			if err := ctx.SetOwnershipReference(cluster, sts); err != nil {
				return err
			}
			ctx.Logger().Info("Creating the bookkeeper statefulset.",
				"StatefulSet.Name", sts.GetName(),
				"StatefulSet.Namespace", sts.GetNamespace())
			if err := ctx.Client().Create(context.TODO(), sts); err != nil {
				return err
			}
			ctx.Logger().Info("StatefulSet creation success.",
				"StatefulSet.Name", sts.GetName(),
				"StatefulSet.Namespace", sts.GetNamespace())
			return nil
		})
}

func shouldUpdateStatefulSet(ctx reconciler.Context, c *v1alpha1.BookkeeperCluster, sts *v1.StatefulSet) bool {
	if *c.Spec.Size != *sts.Spec.Replicas {
		ctx.Logger().Info("Bookkeeper cluster size changed",
			"from", *sts.Spec.Replicas, "to", *c.Spec.Size)
		return true
	}
	if c.Spec.BookkeeperVersion != c.Status.Metadata.BkVersion {
		ctx.Logger().Info("Bookkeeper version changed",
			"from", c.Status.Metadata.BkVersion, "to", c.Spec.BookkeeperVersion,
		)
		return true
	}
	if !mapEqual(c.Spec.BkConfig, c.Status.Metadata.BkConfig) {
		ctx.Logger().Info("Bookkeeper cluster config changed",
			"from", c.Status.Metadata.BkConfig, "to", c.Spec.BkConfig,
		)
		return true
	}
	return false
}

func updateStatefulset(ctx reconciler.Context, sts *v1.StatefulSet, cluster *v1alpha1.BookkeeperCluster) error {
	sts.Spec.Replicas = cluster.Spec.Size
	containers := sts.Spec.Template.Spec.Containers
	for i, container := range containers {
		if container.Name == bookieComponent {
			container.Image = cluster.Image().ToString()
			containers[i] = container
		}
	}
	sts.Spec.Template.Spec.Containers = containers
	ctx.Logger().Info("Updating the bookkeeper statefulset.",
		"StatefulSet.Name", sts.GetName(),
		"StatefulSet.Namespace", sts.GetNamespace(),
		"NewReplicas", cluster.Spec.Size,
		"NewVersion", cluster.Spec.BookkeeperVersion)
	return ctx.Client().Update(context.TODO(), sts)
}

func updateStatefulsetPVCs(ctx reconciler.Context, sts *v1.StatefulSet, cluster *v1alpha1.BookkeeperCluster) error {
	if !cluster.ShouldDeleteStorage() {
		// Keep the orphan PVC since the reclaimed policy said so
		return nil
	}
	pvcList, err := pvc.ListAllWithMatchingLabels(ctx.Client(), sts.Namespace, sts.Spec.Template.Labels)
	if err != nil {
		return err
	}
	for _, item := range pvcList.Items {
		if oputil.IsOrdinalObjectIdle(item.Name, int(*sts.Spec.Replicas)) {
			toDel := &v12.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      item.Name,
					Namespace: item.Namespace,
				},
			}
			ctx.Logger().Info("Deleting the idle pvc. ",
				"StatefulSet.Name", sts.GetName(),
				"StatefulSet.Namespace", sts.GetNamespace(),
				"PVC.Namespace", toDel.GetNamespace(), "PVC.Name", toDel.GetName())
			err = ctx.Client().Delete(context.TODO(), toDel)
			if err != nil {
				return fmt.Errorf("error on deleing the pvc (%s): %w", toDel.Name, err)
			}
		}
	}
	return nil
}

func createStatefulSet(c *v1alpha1.BookkeeperCluster) *v1.StatefulSet {
	labels := c.GenerateWorkloadLabels(bookieComponent)
	return &v1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StatefulSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.GetName(),
			Namespace: c.Namespace,
			Labels: mergeLabels(labels, map[string]string{
				k8s.LabelAppVersion: c.Spec.BookkeeperVersion,
				"version":           c.Spec.BookkeeperVersion,
			}),
			Annotations: c.GenerateAnnotations(),
		},
		Spec: v1.StatefulSetSpec{
			ServiceName: c.HeadlessServiceName(),
			Replicas:    c.Spec.Size,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			UpdateStrategy: v1.StatefulSetUpdateStrategy{
				Type: v1.RollingUpdateStatefulSetStrategyType,
			},
			PodManagementPolicy: v1.OrderedReadyPodManagement,
			Template: v12.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: c.GetName(),
					Labels: mergeLabels(labels,
						c.Spec.PodConfig.Labels,
					),
					Annotations: c.Spec.PodConfig.Annotations,
				},
				Spec: createBookiePodSpec(c),
			},
			VolumeClaimTemplates: createPersistentVolumeClaims(c),
		},
	}
}

func createBookiePodSpec(c *v1alpha1.BookkeeperCluster) v12.PodSpec {
	containerPorts := []v12.ContainerPort{
		{Name: v1alpha1.ClientPortName, ContainerPort: c.Spec.Ports.Bookie},
		{Name: v1alpha1.AdminPortName, ContainerPort: c.Spec.Ports.Admin},
		{Name: v1alpha1.ServiceMetricsPortName, ContainerPort: c.Spec.Ports.Metrics},
	}
	environment := []v12.EnvFromSource{
		{
			ConfigMapRef: &v12.ConfigMapEnvSource{
				LocalObjectReference: v12.LocalObjectReference{
					Name: c.ConfigMapName(),
				},
			},
		},
	}
	image := c.Image()
	volumes := make([]v12.Volume, 0)
	volumeMounts := createVolumeMounts(c.Spec.Directories)
	container := v12.Container{
		Name:      bookieComponent,
		Image:     image.ToString(),
		Ports:     containerPorts,
		EnvFrom:   environment,
		Lifecycle: &v12.Lifecycle{},
		Command: []string{
			"/bin/bash", "/opt/bookkeeper/scripts/entrypoint.sh",
		},
		Args: []string{
			"/opt/bookkeeper/bin/bookkeeper", "bookie",
		},
		Env:             pod.DecorateContainerEnvVars(true, c.Spec.PodConfig.Spec.Env...),
		Resources:       c.Spec.PodConfig.Spec.Resources,
		VolumeMounts:    volumeMounts,
		LivenessProbe:   createLivenessProbe(c.Spec),
		ReadinessProbe:  createReadinessProbe(c.Spec),
		StartupProbe:    createStartupProbe(c.Spec),
		ImagePullPolicy: image.PullPolicy,
	}
	return pod.NewSpec(c.Spec.PodConfig, volumes, nil, []v12.Container{container})
}

func createVolumeMounts(directories *v1alpha1.Directories) []v12.VolumeMount {
	mounts := make([]v12.VolumeMount, 0)
	journalDirs := []string{directories.JournalDir}
	indexDirs := strings.Split(directories.IndexDirs, ",")
	ledgerDirs := strings.Split(directories.LedgerDirs, ",")
	mounts = append(mounts, createVolumeMountsFromDirs(indexDirs, "index")...)
	mounts = append(mounts, createVolumeMountsFromDirs(ledgerDirs, "ledger")...)
	mounts = append(mounts, createVolumeMountsFromDirs(journalDirs, "journal")...)
	return mounts
}

func createVolumeMountsFromDirs(directories []string, volumeName string) []v12.VolumeMount {
	volumeMounts := make([]v12.VolumeMount, 0)
	manyDir := len(directories) > 1
	for i, directory := range directories {
		subPath := ""
		if manyDir {
			subPath = volumeName + strconv.Itoa(i)
		}
		v := v12.VolumeMount{
			Name:      volumeName,
			MountPath: directory,
			SubPath:   subPath,
		}
		volumeMounts = append(volumeMounts, v)
	}
	return volumeMounts
}

func createStartupProbe(spec v1alpha1.BookkeeperClusterSpec) *v12.Probe {
	probe := spec.ProbeConfig.Startup.ToK8sProbe(v12.ProbeHandler{
		HTTPGet: &v12.HTTPGetAction{
			Port: intstr.FromInt32(spec.Ports.Admin),
			Path: "/heartbeat",
		},
	})
	probe.InitialDelaySeconds = probeInitialDelaySeconds
	probe.FailureThreshold = probeFailureThreshold
	return probe
}

func createReadinessProbe(spec v1alpha1.BookkeeperClusterSpec) *v12.Probe {
	probe := spec.ProbeConfig.Readiness.ToK8sProbe(v12.ProbeHandler{
		HTTPGet: &v12.HTTPGetAction{
			Port: intstr.FromInt32(spec.Ports.Admin),
			Path: "/api/v1/bookie/is_ready",
		},
	})
	probe.InitialDelaySeconds = probeInitialDelaySeconds
	probe.FailureThreshold = probeFailureThreshold
	return probe
}

func createLivenessProbe(spec v1alpha1.BookkeeperClusterSpec) *v12.Probe {
	probe := spec.ProbeConfig.Liveness.ToK8sProbe(v12.ProbeHandler{
		HTTPGet: &v12.HTTPGetAction{
			Port: intstr.FromInt32(spec.Ports.Admin),
			Path: "/heartbeat",
		},
	})
	probe.InitialDelaySeconds = probeInitialDelaySeconds
	probe.FailureThreshold = probeFailureThreshold
	return probe
}

func createPersistentVolumeClaims(c *v1alpha1.BookkeeperCluster) []v12.PersistentVolumeClaim {
	persistence := c.Spec.Persistence
	pvcs := []v12.PersistentVolumeClaim{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "index",
				Labels: mergeLabels(
					c.GenerateLabels(),
				),
				Annotations: c.Spec.Persistence.Annotations,
			},
			Spec: *persistence.IndexVolumeClaimSpec,
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ledger",
				Labels: mergeLabels(
					c.GenerateLabels(),
				),
				Annotations: c.Spec.Persistence.Annotations,
			},
			Spec: *persistence.LedgerVolumeClaimSpec,
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "journal",
				Labels: mergeLabels(
					c.GenerateLabels(),
				),
				Annotations: c.Spec.Persistence.Annotations,
			},
			Spec: *persistence.LedgerVolumeClaimSpec,
		},
	}
	return pvcs
}
