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
cd 1-infrastructure
terraform init
terraform apply
```

Alternatively, you can manually create the clusters with gcloud CLI.

To create a hub cluster, run:
```
gcloud container clusters create multikueue-hub \
  --project=${PROJECT_ID} \
  --enable-fleet \
  --region=${LOCATION} \
  --workload-pool=${PROJECT_ID}.svc.id.goog \
  --labels="fleet-clusterinventory-management-cluster=true" \
  --labels="fleet-clusterinventory-namespace=kueue-system"
```

To create a worker cluster, run:
```
gcloud container clusters create multikueue-worker \
  --project=${PROJECT_ID} \
  --region=${LOCATION} \
  --enable-fleet
```

For more details, we documentation for the [ClusterProfile sync feature](https://docs.cloud.google.com/kubernetes-engine/fleet-management/docs/generate-inventory-for-integrations).

## MultiKueue

A Terraform module is provided to install and configure MultiKueue with ClusterProfile API.

Replace the plugin image in multikueue-clusterprofile/2-multikueue/modules/kueue/kueue-patches/patch.yaml with your plugin image. Alternatively, you can install the plugin with `go install`.

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