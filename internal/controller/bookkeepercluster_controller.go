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

package controller

import (
	"context"
	"github.com/monimesl/bookkeeper-operator/api/v1alpha1"
	bookkeepercluster2 "github.com/monimesl/bookkeeper-operator/internal/controller/bookkeepercluster"
	"github.com/monimesl/operator-helper/reconciler"
	v12 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	v13 "k8s.io/api/policy/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	_              reconciler.Context    = &BookkeeperClusterReconciler{}
	_              reconciler.Reconciler = &BookkeeperClusterReconciler{}
	reconcileFuncs                       = []func(ctx reconciler.Context, cluster *v1alpha1.BookkeeperCluster) error{
		bookkeepercluster2.ReconcilePodDisruptionBudget,
		bookkeepercluster2.ReconcileConfigMap,
		bookkeepercluster2.ReconcileServices,
		bookkeepercluster2.ReconcileStatefulSet,
		bookkeepercluster2.ReconcileAutoRecovery,
		bookkeepercluster2.ReconcileClusterStatus,
		bookkeepercluster2.ReconcileFinalizer,
	}
)

// BookkeeperClusterReconciler defines the reconciler to reconcile BookkeeperCluster resources
type BookkeeperClusterReconciler struct {
	reconciler.Context
}

// Configure configures the above BookkeeperClusterReconciler
func (r *BookkeeperClusterReconciler) Configure(ctx reconciler.Context) error {
	r.Context = ctx
	return ctx.NewControllerBuilder().
		For(&v1alpha1.BookkeeperCluster{}).
		Owns(&v13.PodDisruptionBudget{}).
		Owns(&v12.StatefulSet{}).
		Owns(&v12.Deployment{}).
		Owns(&v1.ConfigMap{}).
		Owns(&v1.Service{}).
		Complete(r)
}

// Reconcile handles reconciliation request for BookkeeperCluster instances
func (r *BookkeeperClusterReconciler) Reconcile(_ context.Context, request reconcile.Request) (reconcile.Result, error) {
	cluster := &v1alpha1.BookkeeperCluster{}
	return r.Run(request, cluster, func(_ bool) (err error) {
		for _, fun := range reconcileFuncs {
			if err = fun(r, cluster); err != nil {
				break
			}
		}
		return
	})
}
