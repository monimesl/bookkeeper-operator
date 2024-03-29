#!/bin/bash

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

BK_HOME=/opt/bookkeeper
BINDIR=${BK_HOME}/bin
BOOKKEEPER=${BINDIR}/bookkeeper
SCRIPTS_DIR=${BK_HOME}/scripts

export BOOKKEEPER SCRIPTS_DIR
export PATH=$PATH:/opt/bookkeeper/bin

source /opt/bookkeeper/scripts/init_bookie.sh
init_bookie && /opt/bookkeeper/bin/bookkeeper bookie