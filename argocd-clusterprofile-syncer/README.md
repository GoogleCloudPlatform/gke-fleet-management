# Argo CD ClusterProfile Syncer

## Overview
Argo CD ClusterProfile Syncer is a controller that syncs ClusterProfiles to Argo CD cluster secrets. It watches for changes in the cluster inventory and updates the Argo CD cluster secrets accordingly. For example, when a cluster is added to the inventory, the corresponding Argo CD cluster secret will be automatically generated. This allows Argo CD to seamlessly deploy and manage your applications to all clusters in the cluster inventory.

At the moment, only GKE-Enabled cluster profiles are supported

## Install

### Prerequisites
- A hub cluster. For example, to create a GKE hub cluster, run:
```shell
gcloud container clusters create hub-cluster --region=${LOCATION?} \
  --workload-pool=${PROJECT_ID?}.svc.id.goog
gcloud container fleet memberships update hub-cluster --update-labels="fleet-clusterinventory-management-cluster=true" \
  --location=${LOCATION?}
```
- One or more workload clusters managed by the hub cluster. For example, to create 2 GKE workload clusters, run:
```shell
gcloud container clusters create cluster-1 --enable-fleet --region=${LOCATION?}
gcloud container clusters create cluster-2 --enable-fleet --region=${LOCATION?}
```
- Argo CD installed in the hub cluster. To install Argo CD in the GKE hub cluster above, run:
```shell
gcloud container clusters get-credentials hub-cluster --region=${LOCATION?}
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
```

### Build the syncer image

#### Create an artifacts repository to store the container image for the syncer.

Note: This is a one-time setup.

```shell
gcloud artifacts repositories create multicluster-orchestrator \
    --project=${PROJECT_ID?} \
    --repository-format=docker \
    --location=${LOCATION?}
```

#### Give your default compute service account necessary permissions to view Google Cloud Storage objects

```shell
gcloud projects add-iam-policy-binding projects/${PROJECT_ID?} \
--role="roles/storage.objectViewer" \
--member=serviceAccount:${PROJECT_NUMBER?}-compute@developer.gserviceaccount.com
```

#### Give your node service account necessary permissions to RW images in artifacts repository.

```shell
gcloud projects add-iam-policy-binding projects/${PROJECT_ID?} \
--role="roles/artifactregistry.writer" \
--member=serviceAccount:${PROJECT_NUMBER?}-compute@developer.gserviceaccount.com
```

#### Build and upload images using Cloud Build.

```shell
gcloud builds submit --project=${PROJECT_ID?} \
    --region=${LOCATION?} \
    --config=./cloudbuild.yaml
```

### Deploy

#### Deploy the syncer image

```shell
export PATH_TO_IMAGE=${LOCATION?}-docker.pkg.dev/${PROJECT_ID?}/multicluster-orchestrator/argocd-syncer:latest
envsubst '$PATH_TO_IMAGE' < ./install.yaml | kubectl apply -f -
```

#### Verify that secrets are generated for each workload cluster
```shell
$ kubectl get secrets -n argocd 
NAME                                               TYPE     DATA   AGE
fleet-cluster-inventory.cluster-1-us-central1      Opaque   3      10s
fleet-cluster-inventory.cluster-2-us-central1      Opaque   3      10s
fleet-cluster-inventory.mco-hub-us-central1        Opaque   3      10s
```
