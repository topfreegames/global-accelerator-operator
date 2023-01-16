#!/bin/bash

OPERATION=${1}
MGMTCLUSTER=${2}
REMOTECLUSTER=${3}

install() {
kubectl get --context="${1}" -n global-accelerator-operator-system secret "${2}-kubeconfig" &> /dev/null
if [[ $? -eq 0 ]]; then
  echo "${2}-kubeconfig already created, skipping..."
  exit 0
fi

kubectl --context "$2" -n kube-system get sa global-accelerator-operator &> /dev/null
if [[ $? -ne 0 ]]; then
  kubectl --context "$2" -n kube-system create sa global-accelerator-operator 2> /dev/null
fi

kubectl --context "$2" apply -f "$( dirname -- "$0"; )/global-accelerator-operator-clusterrole.yaml" &> /dev/null

kubectl --context "$2" get clusterrolebinding global-accelerator-operator &> /dev/null
if [[ $? -ne 0 ]]; then
  kubectl --context "$2" create clusterrolebinding global-accelerator-operator --clusterrole=global-accelerator-operator --serviceaccount=kube-system:global-accelerator-operator 2> /dev/null
fi

SECRET=$(kubectl --context="$2" -n kube-system get sa global-accelerator-operator -o json 2> /dev/null | jq -r ".secrets[0].name")
TOKEN=$(kubectl --context="$2" -n kube-system get secret "${SECRET}" -o jsonpath='{.data.token}' 2> /dev/null | base64 --decode)
SERVER=$(kubectl config view | yq ".clusters.[] | select(.name == \"${2}\") | .cluster.server")
CABASE64=$(openssl s_client -showcerts -connect $(echo "$SERVER" | cut -d/ -f3-):443 < /dev/null 2> /dev/null | awk '/1 s:\/CN=kubernetes/,/-----END CERTIFICATE-----/' | tail +3 | base64)
KUBECONFIG=$(export SERVER REMOTECLUSTER=$2 CABASE64 TOKEN; < "$( dirname -- "$0"; )/kubeconfig.tpl" envsubst)

VALUES=( "TOKEN=$TOKEN" "SERVER=$SERVER" "CA=$CABASE64" "KUBECONFIG=$KUBECONFIG" )
for VALUE in "${VALUES[@]}" ; do
    if [ -z "$(echo "${VALUE}" | cut -d= -f2)" ];then
      echo "Something went wrong while populating the core values..."
      echo "${VALUE}"
      exit 1
    fi
done

kubectl get --context="${1}" ns global-accelerator-operator-system &> /dev/null
if [[ $? -ne 0 ]]; then
  kubectl create --context="${1}" ns global-accelerator-operator-system 2> /dev/null
fi

kubectl create --context="${1}" -n global-accelerator-operator-system secret generic "${2}-kubeconfig" --type=cluster.x-k8s.io/secret --from-literal=value="${KUBECONFIG}" 2> /dev/null
}

uninstall() {
  kubectl --context "$2" -n kube-system delete sa global-accelerator-operator 2> /dev/null
  kubectl --context "$2" delete -f "$( dirname -- "$0"; )/global-accelerator-operator-clusterrole.yaml" &> /dev/null
  kubectl --context "$2" delete clusterrolebinding global-accelerator-operator 2> /dev/null
  kubectl delete --context="${1}" -n global-accelerator-operator-system secret "${2}-kubeconfig" 2> /dev/null
  echo "finished cleaning up for ${2}"
}

usage() {
    echo "Remote cluster credentials script: Setup the remote cluster with the needed permissions and create a Secret with the kubeconfig in the Management Cluster"
    echo "Usage:
    "
    echo "To setup the remote cluster and create the Secret with the kubeconfig in the Management Cluster:
    ${0} install <managementClusterName> <remoteClusterName>
    "
    echo "To cleanup the integration:
    ${0} uninstall <managementClusterName> <remoteClusterName>
    "
}

if [[ -z ${OPERATION} ]]; then
    echo "error: You must set an operation"
    usage
    exit 1
elif [[ ${OPERATION} != "install" && ${OPERATION} != "uninstall" ]]; then
    echo "error: Operation must be 'install' or 'uninstall'"
    usage
    exit 1
elif [[ -z ${MGMTCLUSTER} ]]; then
    echo "error: You must set the name of the management cluster"
    usage
    exit 1
elif [[ -z ${REMOTECLUSTER} ]]; then
    echo "error: You must set the name of the remote cluster"
    usage
    exit 1
else
    if [[ ${OPERATION} == "install" ]]; then
        install "${MGMTCLUSTER}" "${REMOTECLUSTER}"
    else
        uninstall "${MGMTCLUSTER}" "${REMOTECLUSTER}"
    fi
fi
exit 0
