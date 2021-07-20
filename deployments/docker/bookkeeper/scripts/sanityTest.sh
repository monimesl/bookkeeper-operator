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

set -x -e -u -m

RETRIES=${1:-1}
INIT_SLEEP=${2:-0}
LOG_FILE=output.log

function waitForSignal() {
  set +e
  PID=$1
  waits=0
  sleep "$INIT_SLEEP"
  while [ $waits -lt "$RETRIES" ]; do
    cat $LOG_FILE
    ps -p "$PID" >/dev/null
    if [[ $? -eq 1 ]]; then
      break # process completes
    fi
    waits=$((waits + 1))
    if [[ $waits -eq "$RETRIES" ]]; then
      break
    fi
    sleep 2
    echo "sanity test still running on wait: $waits" >&2
  done
  cat $LOG_FILE
  # shellcheck disable=SC2002
  cat $LOG_FILE | grep 'Exception in thread \"main\"' >/dev/null
  # shellcheck disable=SC2181
  if [[ $? -eq 0 ]]; then
    set -e
    printf "bookkeeper sanity check failed: \n"
    rm $LOG_FILE
    kill "$PID"
    exit 1
  fi
  set -e
  echo "bookkeeper sanity check passed"
  rm $LOG_FILE
  exit 0
}

/opt/bookkeeper/bin/bookkeeper shell bookiesanity &>$LOG_FILE &
waitForSignal $!
