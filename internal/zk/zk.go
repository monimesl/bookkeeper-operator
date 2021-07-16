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

package zk

import (
	"fmt"
	"github.com/go-zookeeper/zk"
	"github.com/monimesl/bookkeeper-operator/api/v1alpha1"
	"github.com/monimesl/operator-helper/config"
	"strings"
	"time"
)

const (
	// ClusterMetadataParentZNode defines the znode to store metadata for the BookkeeperCluster objects
	ClusterMetadataParentZNode = "/bookkeeper-operator-cluster-metadata"
	updateTimeNode             = "update-time"
	sizeNode                   = "size"
)

type Client struct {
	conn                 *zk.Conn
	requiredNodesCreated bool
}

// UpdateZkClusterMetadata update the metadata of the specified cluster
func UpdateZkClusterMetadata(cluster *v1alpha1.BookkeeperCluster) error {
	if cl, err := NewZkClient(cluster); err != nil {
		return err
	} else {
		defer cl.Close()
		return cl.updateClusterSizeMeta(cluster)
	}
}

// DeleteAllBkZNodes deletes all zNodes created by the zookeeper cluster
func DeleteAllBkZNodes(cluster *v1alpha1.BookkeeperCluster) error {
	if cl, err := NewZkClient(cluster); err != nil {
		return err
	} else {
		defer cl.Close()
		return nil //@Todo do the delete here...
	}
}

//NewZkClient creates a new zookeeper client connected to the specified cluster
func NewZkClient(cluster *v1alpha1.BookkeeperCluster) (*Client, error) {
	address := cluster.Spec.ZookeeperUrl
	c, _, err := zk.Connect([]string{address}, 10*time.Second)
	if err != nil {
		return nil, err
	}
	return &Client{conn: c}, nil
}

func (c *Client) updateClusterSizeMeta(cluster *v1alpha1.BookkeeperCluster) error {
	config.RequireRootLogger().Info("Updating the BookkeeperCluster"+
		" metadata in zookeeper", "cluster", cluster.GetName())
	sizeZNode := clusterSizeNode(cluster)
	updateTimeZNode := clusterUpdateTimeNode(cluster)
	err := c.setNodeData(sizeZNode, []byte(fmt.Sprintf("%d", cluster.Spec.Size)))
	if err != nil {
		return err
	}
	now := time.Now().UnixNano() / int64(time.Millisecond)
	return c.setNodeData(updateTimeZNode, []byte(fmt.Sprintf("%d", now)))
}

// Close closes the zookeeper connection
func (c *Client) Close() {
	config.RequireRootLogger().Info("Closing the zookeeper client")
	c.conn.Close()
}

func clusterSizeNode(cluster *v1alpha1.BookkeeperCluster) string {
	return fmt.Sprintf("%s/%s/%s", ClusterMetadataParentZNode, cluster.GetName(), sizeNode)
}

func clusterUpdateTimeNode(cluster *v1alpha1.BookkeeperCluster) string {
	return fmt.Sprintf("%s/%s/%s", ClusterMetadataParentZNode, cluster.GetName(), updateTimeNode)
}

func (c *Client) createRequiredNodes() (err error) {
	if !c.requiredNodesCreated {
		_ = c.setNodeData(ClusterMetadataParentZNode, nil)
	}
	return
}

func (c *Client) setNodeData(path string, data []byte) (err error) {
	config.RequireRootLogger().
		Info("Creating the operator metadata node",
			"path", path, "data", string(data))
	stats, err := c.getNodeData(path)
	if err == zk.ErrNoNode {
		return c.createNode(path, data)
	} else if err != nil {
		return err
	}
	if _, err = c.conn.Set(path, data, stats.Version); err == zk.ErrNodeExists {
		return nil
	}
	return
}

func (c *Client) getNodeData(clusterNode string) (*zk.Stat, error) {
	_, sts, err := c.conn.Get(clusterNode)
	if err != nil {
		return nil, err
	}
	return sts, nil
}

func (c *Client) createNode(path string, data []byte) error {
	if strings.HasPrefix(path, "/") {
		path = path[1:]
	}
	paths := strings.Split(path, "/")
	zNodes := make([]string, len(paths))
	for i := range paths {
		zNodes[i] = "/" + strings.Join(paths[0:i+1], "/")
	}
	for i, zNode := range zNodes {
		var nodeData []byte = nil
		if i == len(zNodes)-1 {
			// Last node
			nodeData = data
		}
		_, err := c.conn.Create(zNode, nodeData, 0, zk.WorldACL(zk.PermAll))
		if err != nil {
			return err
		}
	}
	return nil
}
