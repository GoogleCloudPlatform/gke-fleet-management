#!/bin/bash
# Usage:
# ./membership-to-clusterprofile.sh $PROJECT $MANAGEMENT_MEMBERSHIP_NAME
# Example:
# ./membership-to-clusterprofile.sh my-project projects/my-project/locations/us-central1/memberships/my-management-membership

# Project to get the memberships from, also used as the namespace name.
project=$1
membership_management_cluster=$2
project_number=$3
namespace=$4
location=$5
echo "using project=$project"
echo "using membership_management_cluster=$membership_management_cluster"
echo "using project_number=$project_number"
echo "using namespace=$namespace"
echo "using location=$location"


# only needed when running outside the cluster
# gcloud container fleet memberships get-credentials  --project $project $membership_management_cluster

function syncClusterProfiles() {
  # used to know the "current" clusterProfiles and delete old ones.
  run_uuid=$(uuidgen)
  echo "new run: $run_uuid, $(date)"
  folder="./inventory-$project-$run_uuid"

  mkdir $folder

  # Get the list of Fleet memberships and write related ClusterProfiles
  # This ignore any health status or type

  while read -r id name version shortName location; do
    echo "Membership name: $name, version: $version, id: $id, location: $location, shortName:$shortName"

    endpointRes=$(curl -s -H "Authorization: Bearer $(gcloud auth print-access-token)" \
      -H "X-GFE-SSL: yes" \
      "https://$location-connectgateway.googleapis.com/v1/projects/$project_number/locations/$location/memberships/$shortName:generateCredentials")
    # echo $endpointRes
    endpoint=$( echo $endpointRes | jq .endpoint)

      read -r -d '' crdContent << EOM
---
apiVersion: multicluster.x-k8s.io/v1alpha1
kind: ClusterProfile
metadata:
  name: $shortName-$location
  namespace: $namespace
  labels:
    x-k8s.io/cluster-manager: fleet-memberships-to-clusterprofile-script
    run_uuid: $run_uuid
    location: $location
  annotations:
    membership-uuid: $id
    argocdns: argocd
spec:
  displayName: $name
  clusterManager:
    name: fleet-memberships-to-clusterprofile-script
EOM

  echo "$crdContent"
  filename="$folder/$shortName-$location.yaml"
  echo "$crdContent" > "$filename"

  done< <(gcloud container fleet memberships list --format="json" --quiet --project $project | jq --raw-output '.[] | "\(.uniqueId) \(.name) \(.endpoint.kubernetesMetadata.kubernetesApiServerVersion) \(.monitoringConfig.cluster) \(.monitoringConfig.location)"')

  # Apply all cluster ClusterProfiles
  kubectl apply -f $folder

  while read -r id name version shortName location; do
    echo "Patching status for $shortName-$location"
    endpointRes=$(curl -s -H "Authorization: Bearer $(gcloud auth print-access-token)" \
      -H "X-GFE-SSL: yes" \
      "https://$location-connectgateway.googleapis.com/v1/projects/$project_number/locations/$location/memberships/$shortName:generateCredentials")
    endpoint=$( echo $endpointRes | jq .endpoint)

    kubectl patch clusterprofile "$shortName-$location" -n "$namespace" --type=merge --subresource=status --patch "
status:
  version:
    kubernetes: $version
  properties:
    - name: clusterset.k8s.io
      value: $project
    - name: location
      value: $location
  credentialProviders:
    - name: gke-connect-gateway
      cluster:
        server: $endpoint"
  done < <(gcloud container fleet memberships list --format="json" --quiet --project $project | jq --raw-output '.[] | "\(.uniqueId) \(.name) \(.endpoint.kubernetesMetadata.kubernetesApiServerVersion) \(.monitoringConfig.cluster) \(.monitoringConfig.location)"')

  rm -rf $folder

  # now delete obsolete memberships (run_uuid not updated)
  while read -r name; do
    kubectl delete clusterprofile $name -n $namespace
    echo "deleted obsolete clusterprofile($name -n $namespace)"
  done< <(kubectl get clusterprofile -n $namespace -o json -l "run_uuid,run_uuid notin ($run_uuid)" | jq --raw-output '.items[] | "\(.metadata.name)"')

  echo "done with run: $run_uuid, $(date)"
}


while true
do
	echo "Press [CTRL+C] to stop.."
  syncClusterProfiles
	sleep 120
done

