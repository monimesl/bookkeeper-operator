[![Language](https://img.shields.io/badge/Language-Go-blue)](https://golang.org/)
[![Go Report Card](https://goreportcard.com/badge/github.com/monimesl/bookkeeper-operator)](https://goreportcard.com/report/github.com/monimesl/bookkeeper-operator)
![Build](https://github.com/monimesl/bookkeeper-operator/workflows/Build/badge.svg)

# Apache Bookkeeper Operator

**Status: *alpha***

Simplify [Bookkeeper](https://bookkeeper.apache.org/) installation and management in kubernetes using CRDs

## Overview

The Bookkeeper Operator enable native [Kubernetes](https://kubernetes.io/)
deployment and management of Apache Bookkeeper cluster. For now,
release [4.11.1](https://bookkeeper.apache.org/docs/4.11.1/overview/overview/) is used as the installed version.

## Prerequisites

The operator needs a kubernetes cluster with a version __>= v1.16.0__ . If you're using [Helm](https://helm.sh/) to
install the operator, your helm version must be __>= 3.0.0__ .

Also, Apache Bookkeeper needs [Zookeeper](https://zookeeper.apache.org/) for metadata storage; if you've not installed
that already, you can use our [Zookeeper Operator](https://github.com/monimesl/zookeeper-operator) to set up an cluster.

## Installation

The operator can be installed and upgrade by using
our [helm chart](https://github.com/monimesl/bookkeeper-operator/tree/main/deployments/charts)
or directly using
the [manifest file](https://github.com/monimesl/bookkeeper-operator/blob/main/deployments/manifest.yaml). We however do
recommend using the [helm chart](https://github.com/monimesl/bookkeeper-operator/tree/main/deployments/charts)
.

### Via [Helm](https://helm.sh/)

First you need to add the chart's [repository](https://monimesl.github.io/helm-charts/) to your repo list:

```bash
helm repo add monimesl https://monimesl.github.io/helm-charts
helm repo update
```

Create the operator namespace; we're doing this because Helm 3 no longer automatically create namespace.

```bash
kubectl create namespace bookkeeper-operator
```

Now install the chart in the created namespace:

```bash
helm install bookkeeper-operator monimesl/bookkeeper-operator -n bookkeeper-operator
```

### Via [Manifest](https://github.com/monimesl/bookkeeper-operator/blob/main/deployments/manifest.yaml)

If you don't have [Helm](https://helm.sh/) or its required version, or you just want to try the operator quickly, this
option is then ideal. We provide a manifest file per operator version. The below command will install the latest
version.

Install the latest tag version:

```bash
 kubectl apply -f https://raw.githubusercontent.com/monimesl/bookkeeper-operator/main/deployments/manifest.yaml
```

Or install the other tagged version you want by using the url below; replace `<tag-here>` with the tag.

```bash
 kubectl apply -f https://raw.githubusercontent.com/monimesl/bookkeeper-operator/<tag-here>/deployments/manifest.yaml
```

Mind you, the command above will install a
[CRD](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)
and create a [ClusterRole](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/); so
the user issuing the command must have cluster-admin privileges.

#### Confirm Installation

Before continuing, ensure the operator pod is __ready__

```bash
kubectl wait --for=condition=ready --timeout=60s pod -l app.kubernetes.io/name=bookkeeper-operator -n bookkeeper-operator
```

When it gets ready, you will see something like this:

```bash
pod/bookkeeper-operator-7975d7d66b-nh2tw condition met
```

If your _wait_ timedout, try another wait.

## Usage

Replace the below `zookeeperUrl` with your own.

#### Creating the simplest Bookkeeper cluster

Apply the following yaml to create the cluster with 3 bookies.

```yaml
apiVersion: bookkeeper.monime.sl/v1alpha1
kind: BookkeeperCluster
metadata:
  name: cluster-1
  namespace: bookkeeper
spec:
  zookeeperUrl: "cluster-1-zk-headless.zookeeper.svc.cluster.local:2181"
  size: 3
```

#### Scale up the cluster from 3 to 5 bookies:

Apply the following yaml to update the `cluster-1` cluster.

```yaml
apiVersion: bookkeeper.monime.sl/v1alpha1
kind: BookkeeperCluster
metadata:
  name: cluster-1
  namespace: bookkeeper
spec:
  zookeeperUrl: "cluster-1-zk-headless.zookeeper.svc.cluster.local:2181"
  size: 5 # scale out
```