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

set -x -e

source /opt/bookkeeper/scripts/common.sh >/dev/null

HOST_IP=$(hostname -i)
ZK_URL=${BK_zkServers:-127.0.0.1:2181}
BOOKIE_ADMIN_PORT=${BK_httpServerPort:-8080}
LEDGERS_ROOT=${BK_zkLedgersRootPath:-"/ledgers"}

ZK_HOST=${ZK_URL%%:*}
ZK_PORT=${ZK_URL##*:}
BK_PORT=${BK_bookiePort:-3181}

export HOST_IP ZK_HOST ZK_PORT BK_PORT BOOKIE_ADMIN_PORT

function waitBookieInit() {
  set +e
  ledgerCreated=false
  retries=0
  while [ $retries -lt 10 ]; do
    sleep 2
    echo "waiting for ledger root: '${LEDGERS_ROOT}' to be created, retry: $retries"
    res=$(zk-shell --run-once "exists $LEDGERS_ROOT" "$ZK_URL")
    nc -z "$ZK_HOST" "$ZK_PORT"
    if echo "$res" | grep -q "czxid"; then
      ledgerCreated=true
      echo "the ledger root: '${LEDGERS_ROOT}' created successfully!"
      break
    fi
    retries=$((retries + 1))
  done
  if [ "$ledgerCreated" == false ]; then
    echo "tired of waiting for the bookie ledger root creation" >&2
    exit 1
  fi
  retries=0
  while [ $retries -lt 100 ]; do
    sleep 2
    echo "waiting for the bookie to be ready, retry: $retries" >&2
    curl "$HOST_IP:$BOOKIE_ADMIN_PORT/api/v1/bookie/is_ready" --fail >/dev/null 2>&1
    # shellcheck disable=SC2181
    if [[ $? -eq 0 ]]; then
      echo "The bookie is ready now!!"
      return
    fi
    retries=$((retries + 1))
  done
  echo "tired of waiting for bookie to be ready" >&2
  exit 1
  set -e
}