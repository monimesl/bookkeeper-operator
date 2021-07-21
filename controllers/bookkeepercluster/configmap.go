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
	"github.com/monimesl/operator-helper/k8s/configmap"
	"github.com/monimesl/operator-helper/oputil"
	"github.com/monimesl/operator-helper/reconciler"
	"github.com/prometheus/common/log"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"strings"
)

// ReconcileConfigMap reconcile the configmap of the specified cluster
func ReconcileConfigMap(ctx reconciler.Context, cluster *v1alpha1.BookkeeperCluster) error {
	cm := &v1.ConfigMap{}
	return ctx.GetResource(types.NamespacedName{
		Name:      cluster.ConfigMapName(),
		Namespace: cluster.Namespace,
	}, cm,
		nil,
		// Not Found
		func() (err error) {
			cm = createConfigMap(cluster)
			if err = ctx.SetOwnershipReference(cluster, cm); err == nil {
				ctx.Logger().Info("Creating the bookkeeper configMap",
					"ConfigMap.Name", cm.GetName(),
					"ConfigMap.Namespace", cm.GetNamespace())
				if err = ctx.Client().Create(context.TODO(), cm); err == nil {
					ctx.Logger().Info("ConfigMap creation success.",
						"ConfigMap.Name", cm.GetName(),
						"ConfigMap.Namespace", cm.GetNamespace())
				}
			}
			return
		})
}

func createConfigMap(cluster *v1alpha1.BookkeeperCluster) *v1.ConfigMap {
	jvmOptions := cluster.Spec.JVMOptions
	excludedOptions := []string{
		"BK_zkServers", "BK_zkLedgersRootPath", "BK_httpServerEnabled", "BK_httpServerPort",
		"BOOKIE_PORT", "BOOKIE_GC_OPTS", "BOOKIE_MEM_OPTS", "BOOKIE_EXTRA_OPTS", "BOOKIE_GC_LOGGING_OPTS",
	}
	data := map[string]string{
		"BK_httpServerEnabled":          "true",
		"BK_useHostNameAsBookieID":      "true",
		"BK_autoRecoveryDaemonEnabled":  "true",
		"BK_lostBookieRecoveryDelay":    "60",
		"BK_zkServers":                  cluster.Spec.ZkServers,
		"BK_CLUSTER_ROOT_PATH":          cluster.ZkRootPath(),
		"BK_zkLedgersRootPath":          cluster.ZkLedgersRootPath(),
		"BK_httpServerPort":             fmt.Sprintf("%d", cluster.Spec.Ports.Admin),
		"BOOKIE_PORT":                   fmt.Sprintf("%d", cluster.Spec.Ports.Bookie),
		"BOOKIE_GC_OPTS":                strings.Join(jvmOptions.Gc, " "),
		"BOOKIE_MEM_OPTS":               strings.Join(jvmOptions.Memory, " "),
		"BOOKIE_EXTRA_OPTS":             strings.Join(jvmOptions.Extra, " "),
		"BOOKIE_GC_LOGGING_OPTS":        strings.Join(jvmOptions.GcLogging, " "),
		"CLUSTER_NAME":                  cluster.GetName(),
		"CLUSTER_METADATA_PARENT_ZNODE": zk.ClusterMetadataParentZNode,
	}
	for k, v := range cluster.Spec.Configs {
		if !strings.HasPrefix(k, "BK_") {
			k = fmt.Sprintf("BK_%s", k)
		}
		if oputil.Contains(excludedOptions, k) {
			log.Warnf("ignoring the config: %s", k)
			continue
		}
		data[k] = v
	}
	return configmap.New(cluster.Namespace, cluster.ConfigMapName(), data)
}
