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
	"github.com/monimesl/operator-helper/k8s/annotation"
	"github.com/monimesl/operator-helper/k8s/pod"
	"github.com/monimesl/operator-helper/k8s/pvc"
	"github.com/monimesl/operator-helper/k8s/statefulset"
	"github.com/monimesl/operator-helper/oputil"
	"github.com/monimesl/operator-helper/reconciler"
	v1 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"strconv"
	"strings"
)

const (
	probeInitialDelaySeconds = 120
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
			if *cluster.Spec.Size != *sts.Spec.Replicas {
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

func updateStatefulset(ctx reconciler.Context, sts *v1.StatefulSet, cluster *v1alpha1.BookkeeperCluster) error {
	sts.Spec.Replicas = cluster.Spec.Size
	ctx.Logger().Info("Updating the bookkeeper statefulset.",
		"StatefulSet.Name", sts.GetName(),
		"StatefulSet.Namespace", sts.GetNamespace(), "NewReplicas", cluster.Spec.Size)
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
				return fmt.Errorf("error on deleing the pvc (%s): %v", toDel.Name, err)
			}
		}
	}
	return nil
}

func createStatefulSet(c *v1alpha1.BookkeeperCluster) *v1.StatefulSet {
	pvcs := createPersistentVolumeClaims(c)
	labels := c.CreateLabels(true, nil)
	templateSpec := createPodTemplateSpec(c, labels)
	spec := statefulset.NewSpec(*c.Spec.Size, c.HeadlessServiceName(), labels, pvcs, templateSpec)
	sts := statefulset.New(c.Namespace, c.StatefulSetName(), labels, spec)
	annotations := c.Spec.Annotations
	if c.Spec.MonitoringConfig.Enabled {
		annotations = annotation.DecorateForPrometheus(
			annotations, true, int(c.Spec.Ports.Metrics))
	}
	sts.Annotations = annotations
	return sts
}

func createPodTemplateSpec(c *v1alpha1.BookkeeperCluster, labels map[string]string) v12.PodTemplateSpec {
	return pod.NewTemplateSpec("", c.StatefulSetName(), labels, c.Spec.PodConfig.Annotations, createPodSpec(c))
}

func createPodSpec(c *v1alpha1.BookkeeperCluster) v12.PodSpec {
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
	init := v12.Container{
		Name:            "bk-init",
		EnvFrom:         environment,
		VolumeMounts:    volumeMounts,
		Image:           image.ToString(),
		ImagePullPolicy: image.PullPolicy,
		Env:             c.Spec.Env,
		Command:         []string{"/bin/sh", "/scripts/init.sh"},
	}
	container := v12.Container{
		Name:            "bk-server",
		EnvFrom:         environment,
		VolumeMounts:    volumeMounts,
		Ports:           containerPorts,
		Image:           image.ToString(),
		ImagePullPolicy: image.PullPolicy,
		Resources:       c.Spec.PodConfig.Resources,
		StartupProbe:    createStartupProbe(c.Spec),
		LivenessProbe:   createLivenessProbe(c.Spec),
		ReadinessProbe:  createReadinessProbe(c.Spec),
		Lifecycle:       &v12.Lifecycle{PreStop: createPreStopHandler()},
		Env:             pod.DecorateContainerEnvVars(true, c.Spec.Env...),
	}
	spec := pod.NewSpec(c.Spec.PodConfig, volumes, []v12.Container{init}, []v12.Container{container})
	spec.TerminationGracePeriodSeconds = c.Spec.PodConfig.TerminationGracePeriodSeconds
	return spec
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

func createPreStopHandler() *v12.Handler {
	return &v12.Handler{Exec: &v12.ExecAction{
		Command: []string{"/bin/sh", "-c", "/scripts/stop.sh"},
	}}
}

func createStartupProbe(spec v1alpha1.BookkeeperClusterSpec) *v12.Probe {
	probe := spec.ProbeConfig.Startup.ToK8sProbe(v12.Handler{
		Exec: &v12.ExecAction{Command: []string{"/scripts/probeStartup.sh"}},
	})
	probe.InitialDelaySeconds = probeInitialDelaySeconds
	return probe
}

func createReadinessProbe(spec v1alpha1.BookkeeperClusterSpec) *v12.Probe {
	probe := spec.ProbeConfig.Readiness.ToK8sProbe(v12.Handler{
		Exec: &v12.ExecAction{Command: []string{"/scripts/probeReadiness.sh"}},
	})
	probe.InitialDelaySeconds = probeInitialDelaySeconds
	return probe
}

func createLivenessProbe(spec v1alpha1.BookkeeperClusterSpec) *v12.Probe {
	probe := spec.ProbeConfig.Liveness.ToK8sProbe(v12.Handler{
		Exec: &v12.ExecAction{Command: []string{"/scripts/probeLiveness.sh"}},
	})
	probe.InitialDelaySeconds = probeInitialDelaySeconds
	return probe
}

func createPersistentVolumeClaims(c *v1alpha1.BookkeeperCluster) []v12.PersistentVolumeClaim {
	return []v12.PersistentVolumeClaim{
		pvc.New(c.Namespace, "index",
			c.CreateLabels(false, nil),
			*c.Spec.Persistence.IndexVolumeClaimSpec,
		),
		pvc.New(c.Namespace, "ledger",
			c.CreateLabels(false, nil),
			*c.Spec.Persistence.LedgerVolumeClaimSpec,
		),
		pvc.New(c.Namespace, "journal",
			c.CreateLabels(false, nil),
			*c.Spec.Persistence.JournalVolumeClaimSpec,
		),
	}
}
