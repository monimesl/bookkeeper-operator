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

# before killing the bookie, we open a temporary tcp listener
# to keep the envoy side container from exiting.
# See https://github.com/istio/istio/issues/7136
nc -l 35316 &
serverPid=$!

killBookie

decommissionBookie

kill $serverPid

echo "Eager kill the process keeping the docker runtime instead of waiting for kubernetes 'TerminationGracePeriodSeconds'"
SLEEP_PROCESS=$(cat sleep.pid)
echo "killing the processing = $SLEEP_PROCESS"
kill "$SLEEP_PROCESS"