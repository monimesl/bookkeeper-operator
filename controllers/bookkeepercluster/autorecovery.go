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
	"github.com/monimesl/bookkeeper-operator/api/v1alpha1"
	"github.com/monimesl/operator-helper/basetype"
	"github.com/monimesl/operator-helper/k8s"
	"github.com/monimesl/operator-helper/k8s/deployment"
	"github.com/monimesl/operator-helper/k8s/pod"
	"github.com/monimesl/operator-helper/reconciler"
	v1 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/core/v1"
	v13 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	autorecoveryComponent = "bookkeeper-autorecovery"
)

// ReconcileAutoRecovery reconcile the deployment of the specified cluster
func ReconcileAutoRecovery(ctx reconciler.Context, cluster *v1alpha1.BookkeeperCluster) error {
	dep := &v1.Deployment{}
	return ctx.GetResource(types.NamespacedName{
		Name:      cluster.AutoRecoveryDeploymentName(),
		Namespace: cluster.Namespace,
	}, dep,
		// Found
		func() error {
			if *cluster.Spec.AutoRecoveryReplicas != *dep.Spec.Replicas {
				return updateAutoRecoveryDeployment(ctx, dep, cluster)
			}
			return nil
		},
		// Not Found
		func() error {
			dep = createAutoRecoveryDeployment(cluster)
			if err := ctx.SetOwnershipReference(cluster, dep); err != nil {
				return err
			}
			ctx.Logger().Info("Creating the bookkeeper deployment.",
				"Deployment.Name", dep.GetName(),
				"Deployment.Namespace", dep.GetNamespace())
			if err := ctx.Client().Create(context.TODO(), dep); err != nil {
				return err
			}
			ctx.Logger().Info("Deployment creation success.",
				"Deployment.Name", dep.GetName(),
				"Deployment.Namespace", dep.GetNamespace())
			return nil
		})
}

func updateAutoRecoveryDeployment(
	ctx reconciler.Context, dep *v1.Deployment,
	cluster *v1alpha1.BookkeeperCluster) error {
	dep.Spec.Replicas = cluster.Spec.AutoRecoveryReplicas
	ctx.Logger().Info("Updating the bookkeeper autorecovery deployment.",
		"Deployment.Name", dep.GetName(),
		"Deployment.Namespace", dep.GetNamespace(), "NewReplicas", cluster.Spec.AutoRecoveryReplicas)
	return ctx.Client().Update(context.TODO(), dep)
}

func createAutoRecoveryDeployment(c *v1alpha1.BookkeeperCluster) *v1.Deployment {
	labels := c.GenerateWorkloadLabels(autorecoveryComponent)
	dep := deployment.New(c.Namespace, c.AutoRecoveryDeploymentName(), labels, v1.DeploymentSpec{
		Replicas: c.Spec.AutoRecoveryReplicas,
		Selector: nil,
		Template: v12.PodTemplateSpec{
			ObjectMeta: pod.NewMetadata(c.Spec.PodConfig, "",
				c.AutoRecoveryDeploymentName(), labels,
				c.GenerateAnnotations()),
			Spec: createAutoRecoveryPodSpec(c),
		},
		Strategy: v1.DeploymentStrategy{},
	})
	dep.Annotations = c.GenerateAnnotations()
	return dep
}

func createAutoRecoveryPodSpec(c *v1alpha1.BookkeeperCluster) v12.PodSpec {
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
	container := v12.Container{
		Name:  autorecoveryComponent,
		Image: image.ToString(),
		Command: []string{
			"/bin/bash", "/opt/bookkeeper/entrypoint.sh",
		},
		Args: []string{
			"/opt/bookkeeper/bin/bookkeeper", "autorecovery",
		},
		EnvFrom:         environment,
		Env:             pod.DecorateContainerEnvVars(true, c.Spec.PodConfig.Spec.Env...),
		ImagePullPolicy: image.PullPolicy,
	}
	spec := pod.NewSpec(basetype.PodConfig{
		ObjectMeta: c.ObjectMeta,
		Spec: basetype.PodSpec{
			Affinity: &v12.Affinity{
				PodAntiAffinity: &v12.PodAntiAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: []v12.PodAffinityTerm{
						{
							LabelSelector: &v13.LabelSelector{
								MatchExpressions: []v13.LabelSelectorRequirement{
									{
										Key:      k8s.LabelAppName,
										Operator: "in",
										Values:   []string{v1alpha1.AppName},
									},
								},
							},
							TopologyKey: "kubernetes.io/hostname",
						},
					},
				},
			},
			ActiveDeadlineSeconds: c.Spec.PodConfig.Spec.ActiveDeadlineSeconds,
			RestartPolicy:         c.Spec.PodConfig.Spec.RestartPolicy,
			ServiceAccountName:    c.Spec.PodConfig.Spec.ServiceAccountName,
			Labels:                c.GenerateWorkloadLabels(autorecoveryComponent),
			NodeName:              c.Spec.PodConfig.Spec.NodeName,
			PriorityClassName:     c.Spec.PodConfig.Spec.PriorityClassName,
			Priority:              c.Spec.PodConfig.Spec.Priority,
			PreemptionPolicy:      c.Spec.PodConfig.Spec.PreemptionPolicy,
		},
	}, volumes, nil, []v12.Container{container})
	return spec
}
