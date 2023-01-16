### Remote cluster credentials script: Setup the remote cluster with the needed permissions and create a Secret with the kubeconfig in the Management Cluster

The goal of this script is to abstract the part of setting up the permissions and credentials of a remote cluster to be monitored. This is only needed to run once for new remote clusters.

It creates a SA `global-accelerator-operator` in the `kube-system` namespace of the remote cluster, then it creates a ClusterRole `global-accelerator-operator` and bind them. After this, it retrieves the needed information to build a kubeconfig to the remote cluster and create a Secret in the namespace `global-accelerator-operator-system` of the Management cluster.

### Requirements

- yq
- openssl
- kubectl
- The remote cluster and management cluster should exist in the kubeconfig context running the script