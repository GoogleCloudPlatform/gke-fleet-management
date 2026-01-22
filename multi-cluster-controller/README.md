# Multi-Cluster Controller Example

This project demonstrates a basic multi-cluster controller built using the [`multicluster-runtime`](https://github.com/kubernetes-sigs/multicluster-runtime) library and the [`ClusterProfile API`](https://github.com/kubernetes-sigs/cluster-inventory-api).

The controller runs in a central hub cluster and is configured to watch for `ConfigMap` changes across multiple remote clusters. It uses a custom GCP authentication plugin and Google Cloud's Workload Identity to securely connect to the remote clusters.

## Overview

The core components of this project are:

*   **Multi-cluster Manager (`cmd/main.go`):** The main controller program that runs in the hub cluster. It uses the [`multicluster-runtime`](https://github.com/kubernetes-sigs/multicluster-runtime) library and the [`cluster-inventory-api`](https://github.com/kubernetes-sigs/multicluster-runtime/tree/main/providers/cluster-inventory-api) provider to watch resources across multiple clusters.
*   **GCP Auth Plugin (`cmd/gcp-auth-plugin/main.go`):** A custom exec plugin that the controller uses to obtain GCP credentials for authenticating to remote GKE clusters.
*   **Kubernetes Manifests (`deploy.yaml`):** Contains the necessary Kubernetes resources to deploy the controller, including the `Deployment`, `ServiceAccount`, `ClusterRole`, and `ClusterRoleBinding`.

## How It Works

1.  The controller is deployed to a central **hub cluster**.
2.  It uses the `ClusterProfile` custom resources, which are automatically generated for each cluster registered to the GKE Fleet, to discover other clusters.
3.  When the controller needs to access a remote cluster, it uses the `gcp-auth-plugin` to obtain temporary credentials via Workload Identity.
4.  The controller then connects to the remote cluster's API server to watch for `ConfigMap` changes.

## Prerequisites

Before you begin, ensure you have the following installed and configured:

*   [Google Cloud SDK](https://cloud.google.com/sdk/docs/install) (`gcloud`)
*   [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
*   [Go](https://golang.org/doc/install)
*   A Google Cloud project with billing enabled.

## Setup and Deployment

### 1. Set Up Your Environment

First, set the following environment variables to simplify the commands in the next steps.

```shell
export PROJECT_ID="your-gcp-project-id"
export PROJECT_NUMBER=$(gcloud projects describe $PROJECT_ID --format="value(projectNumber)")
export LOCATION="us-central1" # Or any other valid GCP region
```

Next, enable the necessary GCP APIs:

```shell
gcloud services enable \
    --project=${PROJECT_ID} \
    container.googleapis.com \
    gkehub.googleapis.com \
    cloudbuild.googleapis.com \
    artifactregistry.googleapis.com
```

### 2. Create GKE Clusters

Create a hub cluster where the controller will run:

```shell
gcloud container clusters create hub-cluster \
  --project=${PROJECT_ID} \
  --enable-fleet \
  --region=${LOCATION} \
  --workload-pool=${PROJECT_ID}.svc.id.goog \
  --labels="fleet-clusterinventory-management-cluster=true"
```

Create two worker clusters that the controller will watch:

```shell
gcloud container clusters create-auto worker-cluster-1 \
  --project=${PROJECT_ID} \
  --region=us-west1 \
  --cluster-version=1.31 \
  --enable-fleet

gcloud container clusters create-auto worker-cluster-2 \
  --project=${PROJECT_ID} \
  --region=us-east1 \
  --cluster-version=1.31 \
  --enable-fleet
```

`ClusterProfile` objects are automatically generated in the hub cluster for all clusters in the Fleet.

### 3. Build and Push the Controller Image

Create an Artifact Registry repository to store the controller's container image:

```shell
gcloud artifacts repositories create multicluster-controller \
    --project=${PROJECT_ID} \
    --repository-format=docker \
    --location=${LOCATION}
```

Build the image using Cloud Build and push it to your new repository:

```shell
gcloud builds submit --project=${PROJECT_ID} \
    --region=${LOCATION} \
    --config=./cloudbuild.yaml
```

### 4. Configure IAM Permissions

The controller's Kubernetes Service Account needs permission to access other clusters via Workload Identity.

Grant the `gkehub.gatewayEditor` and `container.developer` roles to the controller's service account:

```shell
gcloud projects add-iam-policy-binding projects/${PROJECT_ID} \
  --role=roles/gkehub.gatewayEditor \
  --member=principal://iam.googleapis.com/projects/${PROJECT_NUMBER}/locations/global/workloadIdentityPools/${PROJECT_ID}.svc.id.goog/subject/ns/multicluster-controller-system/sa/multicluster-controller-manager

gcloud projects add-iam-policy-binding projects/${PROJECT_ID} \
  --role=roles/container.developer \
  --member=principal://iam.googleapis.googleapis.com/projects/${PROJECT_NUMBER}/locations/global/workloadIdentityPools/${PROJECT_ID}.svc.id.goog/subject/ns/multicluster-controller-system/sa/multicluster-controller-manager
```

### 5. Deploy the Controller

First, get the credentials for your hub cluster:

```shell
gcloud container clusters get-credentials hub-cluster --region=${LOCATION}
```

Next, update the `deploy.yaml` file to use the image you just built. Replace `$PATH_TO_IMAGE` with your image path.

```shell
export IMAGE_PATH="${LOCATION}-docker.pkg.dev/${PROJECT_ID}/multicluster-controller/multi-cluster-controller:latest"
sed -i -e "s|${PATH_TO_IMAGE}|${IMAGE_PATH}|g" deploy.yaml
```

Now, deploy the controller to the hub cluster:

```shell
kubectl apply -f deploy.yaml
```

## Running Locally for Development

For faster iteration, you can run the controller locally. This is useful for testing changes without building and deploying a new image every time.

First, build the auth plugin:
```shell
go build -o gcp-auth-plugin ./cmd/gcp-auth-plugin/main.go
```

Then, run the main controller:

```shell
go run ./cmd/main.go --clusterprofile-provider-file=clusterprofile-provider-file.json
```

## Cleanup

To avoid incurring charges, delete the resources you created.

Delete the GKE clusters:
```shell
gcloud container clusters delete hub-cluster --region=${LOCATION} --project=${PROJECT_ID} --quiet
gcloud container clusters delete worker-cluster-1 --region=us-west1 --project=${PROJECT_ID} --quiet
gcloud container clusters delete worker-cluster-2 --region=us-east1 --project=${PROJECT_ID} --quiet
```

Delete the Artifact Registry repository:
```shell
gcloud artifacts repositories delete multicluster-controller \
    --location=${LOCATION} --project=${PROJECT_ID} --quiet
```

Remove the IAM policy bindings:
```shell
gcloud projects remove-iam-policy-binding projects/${PROJECT_ID} \
  --role=roles/gkehub.gatewayEditor \
  --member=principal://iam.googleapis.com/projects/${PROJECT_NUMBER}/locations/global/workloadIdentityPools/${PROJECT_ID}.svc.id.goog/subject/ns/multicluster-controller-system/sa/multicluster-controller-manager

gcloud projects remove-iam-policy-binding projects/${PROJECT_ID} \
  --role=roles/container.developer \
  --member=principal://iam.googleapis.com/projects/${PROJECT_NUMBER}/locations/global/workloadIdentityPools/${PROJECT_ID}.svc.id.goog/subject/ns/multicluster-controller-system/sa/multicluster-controller-manager
```