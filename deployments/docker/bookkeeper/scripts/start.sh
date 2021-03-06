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

waitZookeeper

deleteCookie

if [ -f bookie_started ]; then
  echo "The previous bookie did not have a clean shutdown"

  decommissionBookie

  rm bookie_started
fi

printf "Starting the bookie in the background.\n"

/scripts/entrypoint.sh bookie &

# wait for the bookie to initialize
waitBookieInit

# perform sanity check on the bookie
# performSanityTest

printf "Probing whether the bookie is alive\n"

netstat -ltn 2>/dev/null | grep "$BK_PORT"

printf "The bookie was successfully started. 👍 \n"

sleep infinity &
PID=$! && JOB=$(jobs -l | grep $PID | cut -d"[" -f2 | cut -d"]" -f1)
echo "$PID" >sleep.pid
echo "true" >bookie_started

fg "$JOB"
