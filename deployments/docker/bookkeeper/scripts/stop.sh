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

set -e -x

source /scripts/common.sh >/dev/null

function killBookie() {
  printf "Stopping the bookie in the background"
  lsof -i :"$BK_PORT" | grep LISTEN | awk '{print $2}' | xargs kill 2>/dev/null
}

function decommissionBookie() {
  set +e
  retries=0
  while [ $retries -lt 4 ]; do
    echo "Decommissioning this bookie with ordinal $MY_ORDINAL from the cluster: $CLUSTER_NAME. retries=$retries"
    /opt/bookkeeper/bin/bookkeeper shell decommissionbookie
    # shellcheck disable=SC2181
    if [[ $? -eq 0 ]]; then
      return
    fi
    retries=$((retries + 1))
    sleep 2
  done
  if [[ "$MY_ORDINAL" -eq "0" ]]; then
    echo "Formatting the bookie with ordinal $MY_ORDINAL"
    /opt/bookkeeper/bin/bookkeeper shell bookieformat -nonInteractive -force -deleteCookie
  fi
  set -e
}

if [ ! -f bookie_started ]; then
    echo "The bookie was never ready, bookie_started file missing"
    return
fi

rm bookie_started ## remove the start indication file

killBookie

decommissionBookie

echo "Eager kill the process keeping the docker runtime instead of waiting for kubernetes 'TerminationGracePeriodSeconds'"
SLEEP_PROCESS=$(cat sleep.pid)
echo "killing the processing = $SLEEP_PROCESS"
kill "$SLEEP_PROCESS"