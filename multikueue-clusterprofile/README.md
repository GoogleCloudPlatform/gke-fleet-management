# MultiKueue with ClusterProfile API and GKE Fleet

In this quickstart, you learn how to set up [MultiKueue][20] with
[ClusterProfile API][21] and [Google Kubernetes Engine (GKE) Fleet][0].

Infrastructure as Code (IaC) is a practice of managing and provisioning software
infrastructure resources using code. Terraform is a popular open source IaC tool
that supports a wide range of Cloud services, including GKE. As a GKE platform
administrator, you can use Terraform to standardize configuration of your
Kubernetes clusters and streamline your DevOps workflows. To learn more, see
[Terraform support for GKE][1].

## Objectives

- Deploy the pre-requisite infrastructure, including:
  - Hub/Manager GKE cluster for MultiKueue
  - Worker GKE clusters in multiple-regions
- Deploy and configure MultiKueue with ClusterProfile API
- Verify the MultiKueue setup

## Before you begin

1. In the Google Cloud console, on the project selector page, select or
[create a Google Cloud project][2].

1. [Make sure that billing is enabled for your Google Cloud project][3].

1. The following service APIs will be enabled:
   1. Cloud Resource Manager
   1. Compute Engine
   1. Google Kubernetes Engine
   1. GKE Hub
   1. Connect Gateway

1. In addition to the `Owner` role, your account will need the following roles:
   1. Connect Gateway Editor (`roles/gkehub.gatewayEditor`)
   1. Kubernetes Engine Developer (`roles/container.developer`)

1. You should be familiar with the basics of Terraform. You can use the following
resources:

- [Getting Started with Terraform][5] (video)
- [Terraform commands][6]

## Prepare the plugin container image

> [!NOTE]
> An official plugin image will be provided in the future, removing the need for this manual build step.

To allow the MultiKueue controller to authenticate to the worker clusters, you need a container image for the credentials plugin. This example uses [gcp-auth-plugin](/gcp-auth-plugin/README.md) in this repository. For detailed instructions on how to build and push the container image to Artifacts Registry, refer to the [documentation](/gcp-auth-plugin/README.md).

## Prepare the environment

In this tutorial, you should use [Cloud Shell][7] to manage resources
hosted on Google Cloud. Cloud Shell is preinstalled with the software you need
for this tutorial, including [Terraform][8], [kubectl][9], and [gcloud CLI][10].

First, export environment variables for your Google Cloud project ID and location:
```
export PROJECT_ID=<project_id>
export LOCATION=<location>
```

> [!IMPORTANT]
> If you do not use Cloud Shell, you may need to install Terraform, kubectl, and
> gcloud CLI. You must also set your default Terraform project with:
> `export GOOGLE_CLOUD_PROJECT=$PROJECT_ID`.

1. Launch a Cloud Shell session from the Google Cloud console, by clicking Cloud
Shell activation icon Activate Cloud Shell in the Google Cloud console. This
launches a session in the bottom pane of the Google Cloud console.

The service credentials associated with this virtual machine are automatic, so
you do not have to set up or download a service account key.

2. Before you run commands, set your default [project][11] in the Google Cloud CLI
  using the following command:

```
gcloud config set project $PROJECT_ID
```

3. Clone the GitHub repository:

```
git clone https://github.com/GoogleCloudPlatform/gke-fleet-management.git --single-branch
```

4. Change to the MultiKueue sample directory:

```
cd gke-fleet-management/multikueue-clusterprofile
```

## Setup

There are two options for deploying the infrastructure and MultiKueue.

### Option 1: Automated setup with Terraform

This option uses Terraform to deploy all the necessary resources.

#### Review the Terraform file

The [Google Cloud Platform Provider][12] is a plugin that lets you
manage and provision Google Cloud resources using Terraform, HashiCorp's
Infrastructure as Code (IaC) tool. The Google Cloud Platform Provider serves as a
bridge between Terraform configurations and the Google Cloud APIs, letting you
define infrastructure resources, such as virtual machines and networks, in a
declarative manner.

1. Review the following Terraform file:

```
cat 1-infrastructure/main.tf
```

The file describes the following resources:

  - IAM permissions
  - GKE hub cluster
  - GKE worker clusters

2. Review the following Terraform file:

```
cat 2-multikueue/main.tf
```

The file describes the following resources:

- A [Helm][15] chart for installing and configuring MultiKueue with
ClusterProfile API

#### Deploy Infrastructure

1. In Cloud Shell, run this command to verify that Terraform is available:

```
terraform version
```

The output should be similar to the following:

```
Terraform v1.10.5
on linux_amd64
```

2. Change Directory:

```
cd 1-infrastructure
```

3. Initialize Terraform:

```
terraform init
```

4. Apply the Terraform configuration:

```
terraform apply
```

Review the plan and when prompted, enter `yes`, to confirm actions.

This command may take around 20 minutes to complete.

#### Deploy MultiKueue

1. Change Directory:

```
cd ../2-multikueue
```

2. Initialize Terraform:

```
terraform init
```

3. After initializing Terraform, you need to replace the placeholder for the plugin image in the `patch.yaml` file. Run the following command:

```shell
sed -i "s|PLUGIN_IMAGE_PLACEHOLDER|${LOCATION}-docker.pkg.dev/${PROJECT_ID}/gcp-auth-plugin/gcp-auth-plugin:v0.0.1|g" modules/kueue/kueue-patches/patch.yaml
```

4. Apply the Terraform configuration:

```
terraform apply
```

When prompted, enter `yes`, to confirm actions.

This command may take a minute to complete.

The output is similar to the following:

```
Apply complete! Resources: 1 added, 0 changed, 0 destroyed.
```

### Option 2: Manual setup with gcloud CLI

This option uses gcloud CLI to manually create the clusters and configure MultiKueue.

#### Deploy Infrastructure

1. Enable the required Google Cloud APIs:
```shell
gcloud services enable \
    cloudresourcemanager.googleapis.com \
    compute.googleapis.com \
    container.googleapis.com \
    gkehub.googleapis.com \
    connectgateway.googleapis.com
```

2. To create a worker cluster, run:
```shell
gcloud container clusters create-auto multikueue-worker \
  --project=${PROJECT_ID} \
  --region=${LOCATION} \
  --enable-fleet
```

Install kueue in the cluster following the [documentation](https://kueue.sigs.k8s.io/docs/installation/#install-a-released-version).

3. To create a hub cluster, run:
```shell
gcloud container clusters create multikueue-hub \
  --project=${PROJECT_ID} \
  --enable-fleet \
  --region=${LOCATION} \
  --workload-pool=${PROJECT_ID}.svc.id.goog \
  --labels=fleet-clusterinventory-management-cluster=true,fleet-clusterinventory-namespace=kueue-system
```

Install kueue in the cluster following the [documentation](https://kueue.sigs.k8s.io/docs/installation/#install-a-released-version).

4. Verify that `ClusterProfile` objects are generated in the hub cluster:
```shell
gcloud container clusters get-credentials multikueue-hub \
  --location=${LOCATION} \
  --project=${PROJECT_ID}

kubectl get clusterprofile -n kueue-system
```

For more details, see the documentation for the [ClusterProfile sync feature][21].

5. Grant the KSA the required IAM roles:
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

#### Deploy MultiKueue

Manually install and configure MultiKueue with ClusterProfile API following the [documentation](https://kueue.sigs.k8s.io/docs/tasks/manage/setup_multikueue/#setup-multikueue-with-clusterprofile-api). Remember to use the plugin you built in the [image preparation section](#prepare-the-plugin-container-image) section.


## Verify the MultiKueue setup is working

Do the following to confirm the MultiKueue is running correctly:

1. Obtain hub cluster credentials:

```
gcloud container clusters get-credentials multikueue-hub --region us-central1
```

2. Apply the sample MultiKueue setup:

> [!NOTE]
> If you set up manually with gcloud, change the `clusterProfileRef` to match your worker clusters.

```
kubectl apply -f ../multikueue-setup.yaml
```

3. The clusters should be active and connected:

```
kubectl get multikueuecluster -n kueue-system
```

The response should be similar to:

```
NAME                             CONNECTED   AGE
multikueue-worker-europe-west4   True        1m
multikueue-worker-us-east1       True        1m
multikueue-worker-us-west1       True        1m
```

## Clean up

To avoid incurring charges to your Google Cloud account for the resources used
on this quickstart, follow these steps.

If you followed the Terraform setup, run the following command first in `2-multikueue`
and then after in `1-infrastructure` to delete the Terraform resources:

```
terraform destroy --auto-approve
```

If you followed the manual setup, delete the projects and clusters you created.

## What's next

* [Deploy a batch system using Kueue][22]

[0]: https://docs.cloud.google.com/kubernetes-engine/docs/fleets-overview
[1]: https://cloud.google.com/kubernetes-engine/docs/resources/use-terraform-gke
[2]: https://cloud.google.com/resource-manager/docs/creating-managing-projects
[3]: https://cloud.google.com/billing/docs/how-to/verify-billing-enabled#console
[5]: https://www.youtube.com/watch?v=BUPenAjobjw
[6]: https://cloud.google.com/docs/terraform/basic-commands
[7]: https://cloud.google.com/shell
[8]: https://cloud.google.com/docs/terraform/get-started-with-terraform
[9]: https://kubernetes.io/docs/reference/kubectl/
[10]: https://cloud.google.com/sdk/gcloud
[11]: https://support.google.com/cloud/answer/6158840
[12]: https://registry.terraform.io/providers/hashicorp/google/latest/docs
[15]: https://helm.sh/
[20]: https://kueue.sigs.k8s.io/docs/tasks/manage/setup_multikueue/
[21]: https://docs.cloud.google.com/kubernetes-engine/fleet-management/docs/generate-inventory-for-integrations
[22]: https://docs.cloud.google.com/kubernetes-engine/docs/tutorials/kueue-intro#create_jobs_and_observe_the_admitted_workloads
