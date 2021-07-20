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
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
)

// ReconcileClusterStatus reconcile the status of the specified cluster
func ReconcileClusterStatus(ctx reconciler.Context, cluster *v1alpha1.BookkeeperCluster) (err error) {
	expectedClusterSize := int(*cluster.Spec.Size)
	labels := cluster.CreateLabels(true, nil)
	err = updateMetadata(ctx, cluster)
	if err != nil {
		return err
	}
	readyReplicas, unreadyReplicas, err := pod.ListAllWithMatchingLabelsByReadiness(ctx.Client(), cluster.Name, labels)
	if err != nil {
		return err
	}
	if expectedClusterSize == len(readyReplicas) {
		cluster.Status.SetPodsReadyConditionTrue()
	} else if len(readyReplicas) == 0 {
		cluster.Status.SetPodsReadyConditionFalse()
	} else {
		cluster.Status.SetPodsReadyConditionFalse()
	}
	var (
		readyMembers   []string
		unreadyMembers []string
	)
	for _, p := range readyReplicas {
		readyMembers = append(readyMembers, p.Name)
	}
	for _, p := range unreadyReplicas {
		unreadyMembers = append(unreadyMembers, p.Name)
	}
	cluster.Status.Membership.Ready = readyMembers
	cluster.Status.Membership.Unready = unreadyMembers
	cluster.Status.ReadyReplicas = int32(len(readyReplicas))
	cluster.Status.CurrentReplicas = int32(len(readyReplicas) + len(unreadyReplicas))
	if err = ctx.Client().Status().Update(context.TODO(), cluster); err != nil {
		err = fmt.Errorf("error on updating the cluster (%s) status: %v", cluster.Name, err)
	}
	return
}

func updateMetadata(ctx reconciler.Context, cluster *v1alpha1.BookkeeperCluster) error {
	if *cluster.Spec.Size != cluster.Status.Metadata.Size {
		ctx.Logger().Info("Setting the cluster metadata",
			"cluster", cluster.Name)
		sts := &v1.StatefulSet{}
		return ctx.GetResource(types.NamespacedName{
			Name:      cluster.StatefulSetName(),
			Namespace: cluster.Namespace,
		}, sts,
			func() (err error) {
				if err = zk.UpdateMetadata(cluster); err == nil {
					cluster.Status.Metadata.Size = *cluster.Spec.Size
					err = ctx.Client().Status().Update(context.TODO(), cluster)
				}
				return
			}, nil)
	}
	return nil
}
