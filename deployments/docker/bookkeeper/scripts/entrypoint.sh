#!/bin/bash

export PATH=$PATH:/opt/bookkeeper/bin

BK_HOME=/opt/bookkeeper
BINDIR=${BK_HOME}/bin
BOOKKEEPER=${BINDIR}/bookkeeper
SCRIPTS_DIR=${BK_HOME}/scripts

if [ $# = 0 ]; then
  echo "No command is found"
  exit 1
fi

COMMAND=$1
shift

function run_command() {
  # shellcheck disable=SC2145
  echo "Run command '$@'"
  "$@"
}

# shellcheck disable=SC1090
source ${SCRIPTS_DIR}/init_"${COMMAND}".sh
init_"${COMMAND}"
run_command ${BOOKKEEPER} "${COMMAND}" "$@"
