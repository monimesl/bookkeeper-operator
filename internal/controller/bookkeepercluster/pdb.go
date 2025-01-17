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
	"github.com/monimesl/operator-helper/k8s"
	"github.com/monimesl/operator-helper/reconciler"
	v1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// ReconcilePodDisruptionBudget reconcile the poddisruptionbudget of the specified cluster
func ReconcilePodDisruptionBudget(ctx reconciler.Context, cluster *v1alpha1.BookkeeperCluster) error {
	return reconcilePodDisruptionBudget(ctx, cluster)
}

func reconcilePodDisruptionBudget(ctx reconciler.Context, cluster *v1alpha1.BookkeeperCluster) (err error) {
	pdb := &v1.PodDisruptionBudget{}
	return ctx.GetResource(types.NamespacedName{
		Name:      cluster.Name,
		Namespace: cluster.Namespace,
	}, pdb,
		func() error {
			// Found
			if shouldUpdatePDB(cluster.Spec, pdb) {
				if err = updatePodDisruptionBudget(ctx, pdb, cluster); err != nil {
					return err
				}
				return nil
			}
			return nil
		},
		// Not Found
		func() error {
			pdb = createPodDisruptionBudget(cluster)
			if err := ctx.SetOwnershipReference(cluster, pdb); err != nil {
				return err
			}
			ctx.Logger().Info("Creating the bookkeeper poddisruptionbudget for cluster",
				"cluster", cluster.Name,
				"PodDisruptionBudget.Name", pdb.GetName(),
				"PodDisruptionBudget.Namespace", pdb.GetNamespace(),
				"MaxUnavailable", pdb.Spec.MaxUnavailable.IntVal)
			return ctx.Client().Create(context.TODO(), pdb)
		},
	)
}

func createPodDisruptionBudget(cluster *v1alpha1.BookkeeperCluster) *v1.PodDisruptionBudget {
	maxFailureNodes := intstr.FromInt32(cluster.Spec.MaxUnavailableNodes)
	return &v1.PodDisruptionBudget{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PodDisruptionBudget",
			APIVersion: "policy/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name,
			Namespace: cluster.Namespace,
			Labels:    cluster.GenerateLabels(),
		},
		Spec: v1.PodDisruptionBudgetSpec{
			MaxUnavailable: &maxFailureNodes,
			Selector: &metav1.LabelSelector{
				MatchLabels: cluster.GenerateWorkloadLabels(bookieComponent),
			},
		},
	}
}

func updatePodDisruptionBudget(ctx reconciler.Context, pdb *v1.PodDisruptionBudget, c *v1alpha1.BookkeeperCluster) error {
	newMaxFailureNodes := intstr.FromInt32(c.Spec.MaxUnavailableNodes)
	pdb.Labels = c.GenerateLabels()
	pdb.Spec.MaxUnavailable.IntVal = newMaxFailureNodes.IntVal
	pdb.Spec.Selector.MatchLabels = c.GenerateWorkloadLabels(bookieComponent)
	ctx.Logger().Info("Updating the bookkeeper poddisruptionbudget for cluster",
		"cluster", c.Name,
		"PodDisruptionBudget.Name", pdb.GetName(),
		"PodDisruptionBudget.Namespace", pdb.GetNamespace(),
		"MaxUnavailable", pdb.Spec.MaxUnavailable.IntVal)
	return ctx.Client().Update(context.TODO(), pdb)
}

func shouldUpdatePDB(spec v1alpha1.BookkeeperClusterSpec, pdb *v1.PodDisruptionBudget) bool {
	if spec.BookkeeperVersion != pdb.Labels[k8s.LabelAppVersion] {
		return true
	}
	newMaxFailureNodes := intstr.FromInt32(spec.MaxUnavailableNodes)
	return newMaxFailureNodes.IntVal != pdb.Spec.MaxUnavailable.IntVal
}
