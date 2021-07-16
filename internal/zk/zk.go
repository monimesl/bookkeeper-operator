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
	"github.com/go-zookeeper/zk"
	"github.com/monimesl/bookkeeper-operator/api/v1alpha1"
	"github.com/monimesl/operator-helper/config"
	"time"
)

type Client struct {
	conn *zk.Conn
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

//NewZkClient creates a new zookeeper client connected to the zookeeper cluster specified in BookkeeperCluster
func NewZkClient(cluster *v1alpha1.BookkeeperCluster) (*Client, error) {
	address := cluster.Spec.ZookeeperUrl
	c, _, err := zk.Connect([]string{address}, 10*time.Second)
	if err != nil {
		return nil, err
	}
	return &Client{conn: c}, nil
}

// Close closes the zookeeper connection
func (c *Client) Close() {
	config.RequireRootLogger().Info("Closing the zookeeper client")
	c.conn.Close()
}
