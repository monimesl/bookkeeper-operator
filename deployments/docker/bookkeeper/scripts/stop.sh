#!/usr/bin/env bash

#
# Copyright 2021 - now, the original author or authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#       https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

source /scripts/common.sh

set -e -x -m

function killBookie() {
  printf "Stopping the bookie in the background.\n"
  lsof -i :"$BK_PORT" | grep LISTEN | awk '{print $2}' | xargs kill 2>/dev/nul
}

function maybeDecommissionBookie() {
  set +e
  echo "Syncing and fetching the size of the cluster $CLUSTER_NAME"
  SIZE=""
  for ((i = 0; i < 15; i++)); do
    SYNC=$(zk-shell "$ZK_URL" --run-once "sync $CLUSTER_META_NODE_PATH")
    if [[ -z "${SYNC}" ]]; then
      SIZE=$(zk-shell "$ZK_URL" --run-once "get $CLUSTER_META_NODE_PATH" | cut -d"=" -f2)
      break
    fi
    echo "Failed to connect. Retrying($i) after 2 seconds"
    SIZE=""
    sleep 2
  done
  echo "Cluster current SIZE=$SIZE, myid=$MY_ORDINAL"

  # Since we're using kubernetes statefulset to start the bookie in an ordered fashion,
  # the cluster size at any arbitrary normal point in time equals the highest `myid`.
  # which is 1 increment of the ordinal of the pod running the container. On cluster
  # down scaling($SIZE reduction), the pod with the highest ordinal hence `myid` is deleted.
  # This means any bookie whose `myid` is greater than the current cluster size is being
  # permanently removed from the ensemble
  if [[ -n "$SIZE" && "$MY_ORDINAL" -gt "$SIZE" ]]; then
    echo "Decommissioning this bookie with ordinal $MY_ORDINAL from the cluster: $CLUSTER_NAME"
    /opt/bookkeeper/bin/bookkeeper shell decommissionbookie
  fi
  set -e

}

killBookie

sleep 2

maybeDecommissionBookie

echo "Eager kill the process keeping the docker runtime instead of waiting for kubernetes 'TerminationGracePeriodSeconds'"

SLEEP_PROCESS=$(cat sleep.pid)

kill "$SLEEP_PROCESS"
