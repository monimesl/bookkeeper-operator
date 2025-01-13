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
	"github.com/monimesl/bookkeeper-operator/internal/zk"
	"github.com/monimesl/operator-helper/k8s/pod"
	"github.com/monimesl/operator-helper/reconciler"
)

// ReconcileClusterStatus reconcile the status of the specified cluster
//
//nolint:nakedret
func ReconcileClusterStatus(ctx reconciler.Context, cluster *v1alpha1.BookkeeperCluster) (err error) {
	err = updateMetadata(ctx, cluster)
	if err != nil {
		return err
	}
	labels := cluster.GenerateLabels()
	readyReplicas, unreadyReplicas, err := pod.ListAllWithMatchingLabelsByReadiness(ctx.Client(), cluster.Name, labels)
	if err != nil {
		return err
	}
	expectedClusterSize := int(*cluster.Spec.Size)
	switch {
	case expectedClusterSize == len(readyReplicas):
		cluster.Status.SetPodsReadyConditionTrue()
	case len(readyReplicas) == 0:
		cluster.Status.SetPodsReadyConditionFalse()
	default:
		cluster.Status.SetPodsReadyConditionFalse()
	}
	readyMembers := make([]string, len(readyReplicas))
	for i, p := range readyReplicas {
		readyMembers[i] = p.Name
	}
	unreadyMembers := make([]string, len(unreadyReplicas))
	for i, p := range unreadyReplicas {
		unreadyMembers[i] = p.Name
	}
	cluster.Status.Membership.Ready = readyMembers
	cluster.Status.Membership.Unready = unreadyMembers
	cluster.Status.ReadyReplicas = int32(len(readyReplicas))
	cluster.Status.CurrentReplicas = int32(len(readyReplicas) + len(unreadyReplicas))
	if err = ctx.Client().Status().Update(context.TODO(), cluster); err != nil {
		err = fmt.Errorf("error on updating the cluster (%s) status: %w", cluster.Name, err)
	}
	return
}

func updateMetadata(ctx reconciler.Context, c *v1alpha1.BookkeeperCluster) error {
	if *c.Spec.Size != c.Status.Metadata.Size ||
		!mapEqual(c.Spec.BkConfig, c.Status.Metadata.BkConfig) ||
		c.Spec.BookkeeperVersion != c.Status.Metadata.BkVersion {
		ctx.Logger().Info("Reconciling the cluster status data",
			"cluster", c.GetName(), "deletionTimestamp", c.DeletionTimestamp,
			"specSize", c.Spec.Size, "specVersion", c.Spec.BookkeeperVersion, "specConfig", c.Spec.BkConfig,
			"status", c.Status)
		// Update metadata only if the cluster is not being deleted
		if c.DeletionTimestamp.IsZero() {
			c.Status.Metadata.BkConfig = c.Spec.BkConfig
			c.Status.Metadata.BkVersion = c.Spec.BookkeeperVersion
			if *c.Spec.Size != c.Status.Metadata.Size {
				ctx.Logger().Info("Updating the cluster status bookkeeper metadata",
					"cluster", c.GetName(), "specSize", *c.Spec.Size,
					"statusSize", c.Status.Metadata.Size)
				if err := zk.UpdateMetadata(c); err != nil {
					return err
				}
			}
			c.Status.Metadata.Size = *c.Spec.Size
			ctx.Logger().Info("Updating the cluster status", "cluster", c.GetName(), "status", c.Status)
			if err := ctx.Client().Status().Update(context.TODO(), c); err != nil {
				ctx.Logger().Info("Error updating the cluster status", "error", err)
				return err
			}
		}
	}
	return nil
}
