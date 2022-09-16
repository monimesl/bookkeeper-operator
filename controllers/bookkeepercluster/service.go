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
	"github.com/monimesl/operator-helper/k8s/service"
	"github.com/monimesl/operator-helper/reconciler"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// ReconcileServices reconcile the services of the specified cluster
func ReconcileServices(ctx reconciler.Context, cluster *v1alpha1.BookkeeperCluster) (err error) {
	if err = reconcileHeadlessService(ctx, cluster); err == nil {
		err = reconcileClientService(ctx, cluster)
	}
	return
}

func reconcileClientService(ctx reconciler.Context, cluster *v1alpha1.BookkeeperCluster) error {
	svc := &v1.Service{}
	return ctx.GetResource(types.NamespacedName{
		Name:      cluster.ClientServiceName(),
		Namespace: cluster.Namespace,
	}, svc,
		nil,
		// Not Found
		func() (err error) {
			svc = createClientService(cluster)
			if err = ctx.SetOwnershipReference(cluster, svc); err == nil {
				ctx.Logger().Info("Creating the bookkeeper client service.",
					"Service.Name", svc.GetName(),
					"Service.Namespace", svc.GetNamespace())
				if err = ctx.Client().Create(context.TODO(), svc); err == nil {
					ctx.Logger().Info("Service creation success.",
						"Service.Name", svc.GetName(),
						"Service.Namespace", svc.GetNamespace())
				}
			}
			return
		})
}

func reconcileHeadlessService(ctx reconciler.Context, cluster *v1alpha1.BookkeeperCluster) error {
	svc := &v1.Service{}
	return ctx.GetResource(types.NamespacedName{
		Name:      cluster.HeadlessServiceName(),
		Namespace: cluster.Namespace,
	}, svc,
		nil,
		// Not Found
		func() (err error) {
			svc = createHeadlessService(cluster)
			svc.Spec.PublishNotReadyAddresses = true
			if err = ctx.SetOwnershipReference(cluster, svc); err == nil {
				ctx.Logger().Info("Creating the bookkeeper headless service.",
					"Service.Name", svc.GetName(),
					"Service.Namespace", svc.GetNamespace())
				if err = ctx.Client().Create(context.TODO(), svc); err == nil {
					ctx.Logger().Info("Service creation success.",
						"Service.Name", svc.GetName(),
						"Service.Namespace", svc.GetNamespace())
				}
			}
			return
		})
}

func createClientService(c *v1alpha1.BookkeeperCluster) *v1.Service {
	return createService(c, c.ClientServiceName(), true, servicePorts(c))
}

func createHeadlessService(c *v1alpha1.BookkeeperCluster) *v1.Service {
	return createService(c, c.HeadlessServiceName(), false, servicePorts(c))
}

func createService(c *v1alpha1.BookkeeperCluster, name string, hasClusterIp bool, servicePorts []v1.ServicePort) *v1.Service {
	labels := c.GenerateWorkloadLabels(bookieComponent)
	clusterIp := ""
	if !hasClusterIp {
		clusterIp = v1.ClusterIPNone
	}
	srv := service.New(c.Namespace, name, labels, v1.ServiceSpec{
		ClusterIP: clusterIp,
		Selector:  labels,
		Ports:     servicePorts,
	})
	srv.Annotations = c.GenerateAnnotations()
	return srv
}

func servicePorts(c *v1alpha1.BookkeeperCluster) []v1.ServicePort {
	return []v1.ServicePort{
		{Name: v1alpha1.ClientPortName, Port: c.Spec.Ports.Bookie},
		{Name: v1alpha1.AdminPortName, Port: c.Spec.Ports.Admin},
		{Name: v1alpha1.ServiceMetricsPortName, Port: c.Spec.Ports.Metrics},
	}
}
