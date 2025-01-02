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
	"github.com/monimesl/operator-helper/config"
	"github.com/monimesl/operator-helper/k8s/configmap"
	"github.com/monimesl/operator-helper/oputil"
	"github.com/monimesl/operator-helper/reconciler"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"strconv"
	"strings"
)

// ReconcileConfigMap reconcile the configmap of the specified cluster
func ReconcileConfigMap(ctx reconciler.Context, cluster *v1alpha1.BookkeeperCluster) error {
	cm := &v1.ConfigMap{}
	return ctx.GetResource(types.NamespacedName{
		Name:      cluster.ConfigMapName(),
		Namespace: cluster.Namespace,
	}, cm,
		// Found
		func() error {
			if err := updateConfigmap(ctx, cm, cluster); err != nil {
				return err
			}
			return nil
		},
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

func createConfigMap(c *v1alpha1.BookkeeperCluster) *v1.ConfigMap {
	data := createConfigmapData(c)
	cm := configmap.New(c.Namespace, c.ConfigMapName(), data)
	cm.Labels = c.GenerateLabels()
	return cm
}

func updateConfigmap(ctx reconciler.Context, cm *v1.ConfigMap, c *v1alpha1.BookkeeperCluster) error {
	ctx.Logger().Info("Updating the bookkeeper configmap.",
		"configMap.Name", cm.GetName(),
		"ConfigMap.Namespace", cm.GetNamespace(), "NewReplicas", c.Spec.Size)
	cm.Labels = c.GenerateLabels()
	cm.Data = createConfigmapData(c)
	return ctx.Client().Update(context.TODO(), cm)
}

func createConfigmapData(c *v1alpha1.BookkeeperCluster) map[string]string {
	jvmOptions := c.Spec.JVMOptions
	excludedOptions := []string{
		"BK_zkServers", "BK_zkLedgersRootPath", "BK_httpServerEnabled", "BK_httpServerPort", "BK_enableStatistics",
		"BOOKIE_PORT", "BOOKIE_GC_OPTS", "BOOKIE_MEM_OPTS", "BOOKIE_EXTRA_OPTS", "BOOKIE_GC_LOGGING_OPTS",
		"BK_journalDirectories", "BK_ledgerDirectories", "BK_indexDirectories",
	}
	autoRecovery := true
	if c.Spec.EnableAutoRecovery != nil {
		autoRecovery = *c.Spec.EnableAutoRecovery
	}
	data := map[string]string{
		"BK_enableStatistics":           "true",
		"BK_httpServerEnabled":          "true",
		"BK_useHostNameAsBookieID":      "true",
		"BK_lostBookieRecoveryDelay":    "60",
		"BK_prometheusStatsHttpAddress": "0.0.0.0",
		"BK_CLUSTER_ROOT_PATH":          c.ZkRootPath(),
		"BK_zkServers":                  c.Spec.ZkServers,
		"BK_zkLedgersRootPath":          c.ZkLedgersRootPath(),
		"BK_indexDirectories":           c.Spec.Directories.IndexDirs,
		"BK_ledgerDirectories":          c.Spec.Directories.LedgerDirs,
		"BK_journalDirectories":         c.Spec.Directories.JournalDir,
		"BK_autoRecoveryDaemonEnabled":  strconv.FormatBool(autoRecovery),
		"BK_httpServerPort":             fmt.Sprintf("%d", c.Spec.Ports.Admin),
		"BK_prometheusStatsHttpPort":    fmt.Sprintf("%d", c.Spec.Ports.Metrics),
		"BK_BOOKIE_PORT":                fmt.Sprintf("%d", c.Spec.Ports.Bookie),
		"BK_statsProviderClass":         "org.apache.bookkeeper.stats.prometheus.PrometheusMetricsProvider",
		// https://github.com/apache/bookkeeper/blob/2346686c3b8621a585ad678926adf60206227367/bin/common.sh#L118
		"BK_BOOKIE_MEM_OPTS": strings.Join(jvmOptions.Memory, " "),
		// https://github.com/apache/bookkeeper/blob/2346686c3b8621a585ad678926adf60206227367/bin/common.sh#L119
		"BK_BOOKIE_GC_OPTS": strings.Join(jvmOptions.Gc, " "),
		// https://github.com/apache/bookkeeper/blob/2346686c3b8621a585ad678926adf60206227367/bin/common.sh#L120
		"BK_BOOKIE_GC_LOGGING_OPTS": strings.Join(jvmOptions.GcLogging, " "),
		// https://github.com/apache/bookkeeper/blob/2346686c3b8621a585ad678926adf60206227367/bin/bookkeeper#L149
		"BK_BOOKIE_EXTRA_OPTS": strings.Join(jvmOptions.Extra, " "),
		"CLUSTER_NAME":         c.GetName(),
	}
	for k, v := range c.Spec.BkConfig {
		if !strings.HasPrefix(k, "BK_") {
			k = fmt.Sprintf("BK_%s", k)
		}
		if oputil.Contains(excludedOptions, k) {
			config.RequireRootLogger().Info("ignoring the config", "config", k)
			continue
		}
		data[k] = v
	}
	return data
}
