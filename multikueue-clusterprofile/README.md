# MultiKueue with ClusterProfile API

This directory provides an example for MultiKueue with ClusterProfile API & GKE Fleet.

## Prerequisites

Before you begin, ensure you have the following tools installed:

- [Google Cloud SDK](https://cloud.google.com/sdk/docs/install)
- [Terraform](https://learn.hashicorp.com/tutorials/terraform/install-cli)
- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)

## Infrastructure

A Terraform module is provided to create a hub cluster and three worker clusters.

```shell
export GOOGLE_CLOUD_PROJECT=`gcloud config get-value project`
cd 1-infrastructure
terraform init
terraform apply
```

Alternatively, you can manually create the clusters with gcloud CLI.

### Worker clusters
To create a worker cluster, run:
```shell
gcloud container clusters create-auto multikueue-worker \
  --project=${PROJECT_ID} \
  --region=${LOCATION} \
  --enable-fleet
```

### Hub cluster
To create a hub cluster, run:
```shell
gcloud container clusters create multikueue-hub \
  --project=${PROJECT_ID} \
  --enable-fleet \
  --region=${LOCATION} \
  --workload-pool=${PROJECT_ID}.svc.id.goog \
  --labels="fleet-clusterinventory-management-cluster=true" \
  --labels="fleet-clusterinventory-namespace=kueue-system"
```

Verify that `ClusterProfile` objects are generated in the hub cluster:
```shell
gcloud container clusters get-credentials multikueue-hub \
  --location=${LOCATION} \
  --project=${PROJECT_ID}

kubectl get clusterprofile -n kueue-system
```

For more details, see the documentation for the [ClusterProfile sync feature](https://docs.cloud.google.com/kubernetes-engine/fleet-management/docs/generate-inventory-for-integrations).

### Add IAM policy bindings
Grant the KSA the required IAM roles:
```shell
PROJECT_ID=`gcloud config get-value project`
PROJECT_NUMBER=`gcloud projects describe "${PROJECT_ID}" --format "value(projectNumber)"`
gcloud projects add-iam-policy-binding projects/${PROJECT_ID} --condition=None \
--role=roles/gkehub.gatewayEditor \
--member=principal://iam.googleapis.com/projects/${PROJECT_NUMBER}/locations/global/workloadIdentityPools/${PROJECT_ID?}.svc.id.goog/subject/ns/kueue-system/sa/kueue-controller-manager

gcloud projects add-iam-policy-binding projects/${PROJECT_ID} --condition=None \
--role=roles/container.developer \
--member=principal://iam.googleapis.com/projects/${PROJECT_NUMBER}/locations/global/workloadIdentityPools/${PROJECT_ID?}.svc.id.goog/subject/ns/kueue-system/sa/kueue-controller-manager
```

## MultiKueue

A Terraform module is provided to install and configure MultiKueue with ClusterProfile API.

Replace the plugin image in `multikueue-clusterprofile/2-multikueue/modules/kueue/kueue-patches/patch.yaml` with your plugin image. You can build the plugin image following the [documentation](/gcp-auth-plugin/README.md).

```shell
cd 2-multikueue
terraform init
terraform apply
```

Alternatively, you can manually install and configure MultiKueue with ClusterProfile API following the [documentation](https://kueue.sigs.k8s.io/docs/tasks/manage/setup_multikueue/#setup-multikueue-with-clusterprofile-api).

## Sample MultiKueue setup

In the hub cluster, apply the following config:

```
kubectl apply -f multikueue-setup.yaml
```

The clusters should be active and connected.

```
kubectl get multikueuecluster -n kueue-system
NAME                             CONNECTED   AGE
multikueue-worker-europe-west4   True        13h
multikueue-worker-us-east1       True        13h
multikueue-worker-us-west1       True        13h
```

## Next steps
* [Deploy a batch system using Kueue](https://docs.cloud.google.com/kubernetes-engine/docs/tutorials/kueue-intro#create_jobs_and_observe_the_admitted_workloads)