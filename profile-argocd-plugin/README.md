# ArgoCD Cluster Profile Plugin

## Overview

An example application which users can copy and deploy as an
[ArgoCD generator plugin](https://argo-cd.readthedocs.io/en/stable/operator-manual/applicationset/Generators-Plugin/#plugin-generator)
for clusters with ClusterProfiles.

*   Disclaimer: This is not an official Google product
*   Created by:
    [MCO](https://github.com/GoogleCloudPlatform/gke-fleet-management/tree/main/multi-cluster-orchestrator)
    team at Google

### Setup

For this guide we will use GKE clusters, so that we can use the Argo
ClusterProfile Syncer to generate ClusterProfiles and Secrets.

-   Clone this repo (`gke-fleet-management`)
-   Install the [Google Cloud CLI](https://cloud.google.com/sdk/docs/install)
    (gcloud)

Set env variables:

```shell
PROJECT_ID="${PROJECT_ID:-$(gcloud config get-value project)}"
PROJECT_NUMBER="${PROJECT_NUMBER:-$(gcloud projects describe ${PROJECT_ID} --format="value(projectNumber)")}"
LOCATION=us-central1
```

#### Set up the clusters

Follow the setup for the
[Argo ClusterProfile Syncer](https://github.com/GoogleCloudPlatform/gke-fleet-management/tree/main/argocd-clusterprofile-syncer),
also included in the `gke-fleet-management` repo. If prompted during
`add-iam-policy-binding` commands, select `condition=None`.

Then, set up Argo permissions and the destination namespace in the managed
cluster(s):

```shell
gcloud container clusters get-credentials cluster-1 --location=${LOCATION}
kubectl create namespace webserver
kubectl create namespace argocd-manager
kubectl create serviceaccount argocd-manager-sa -n argocd-manager
kubectl apply -f manager-role.yaml

gcloud container clusters get-credentials cluster-2 --location=${LOCATION}
kubectl create namespace webserver
kubectl create namespace argocd-manager
kubectl create serviceaccount argocd-manager-sa -n argocd-manager
kubectl apply -f manager-role.yaml
```

### Build the plugin

#### Create an Artifact Registry repo to store the container image.

```shell
gcloud artifacts repositories create argocd-profile-plugin \
    --project=${PROJECT_ID} \
    --repository-format=docker \
    --location=${LOCATION} \
    --description="Docker repository for ArgoCD profile plugin"
```

#### Build and upload the plugin image with Cloud Build.

Grant the default service account Cloud Build permissions (if prompted, select
`condition=None`):

```shell
gcloud projects add-iam-policy-binding ${PROJECT_ID} \
    --member="serviceAccount:${PROJECT_NUMBER}-compute@developer.gserviceaccount.com" \
    --role="roles/cloudbuild.builds.builder"
gcloud builds submit --region=us-central1 --config=cloudbuild.yaml
```

You may confirm the image upload in the Google Cloud Console. Should you make
changes to this plugin's code, reload by re-submitting the build and deleting
the `argocd-profile-plugin-[...]` pod in namespace `argocd`.

#### Setup GKE Workload identity federation

This example uses
[GKE workload identity](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity)
federation to authenticate to GCP APIs.

*   If it not already created with WI, enable workload identity federation on
    the hub cluster and its node pools
    ([guide](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity#enable-existing-cluster)).

*   Give the ArgoCD application controller permissions to access clusters with
    GKE Connect. If prompted for `add-iam-policy-binding` commands, select
    `condition=None`.

    ```shell
    # Create Google Service Account
    GSA_NAME=argocd-plugin-controller
    gcloud iam service-accounts create ${GSA_NAME} \
    --project=${PROJECT_ID} \
    --display-name="Argo CD Controller GSA"

    # Grant GSA permissions
    GSA_EMAIL=${GSA_NAME}@${PROJECT_ID}.iam.gserviceaccount.com
    gcloud projects add-iam-policy-binding ${PROJECT_ID} \
    --member="serviceAccount:${GSA_EMAIL}" \
    --role="roles/container.developer"
    gcloud projects add-iam-policy-binding ${PROJECT_ID} \
    --member="serviceAccount:${GSA_EMAIL}" \
    --role="roles/gkehub.gatewayAdmin"

    # Allow Argo Application service account to impersonate GSA
    gcloud iam service-accounts add-iam-policy-binding ${GSA_EMAIL} \
    --project=${PROJECT_ID} \
    --role="roles/iam.workloadIdentityUser" \
    --member="serviceAccount:${PROJECT_ID}.svc.id.goog[argocd/argocd-application-controller]"

    # Annotate the Application KSA with its impersonated GSA
    kubectl annotate serviceaccount argocd-application-controller \
    --namespace argocd \
    "iam.gke.io/gcp-service-account=${GSA_EMAIL}" \
    --overwrite

    # Restart Argo
    kubectl rollout restart deployment -n argocd -l app.kubernetes.io/part-of=argocd
    ```

### Deploy

#### Install the plugin on the control cluster

In `profile-plugin-install.yaml`, fill in the image URI and project number:

```shell
PATH_TO_IMAGE=$(gcloud artifacts repositories describe argocd-profile-plugin --location=${LOCATION} --format="value(registryUri)")
sed -i "s#\$PATH_TO_IMAGE#${PATH_TO_IMAGE}#g" profile-plugin-install.yaml
sed -i "s#\$PROJECT_NUMBER#${PROJECT_NUMBER}#g" profile-plugin-install.yaml
kubectl apply -f profile-plugin-install.yaml
```

If you are using a namespace for ClusterProfiles other than
`fleet-cluster-inventory`, replace it in the file. If you plan to use multiple
namespaces, copy the Role and RoleBinding for each one.

Finally, create an ApplicationSet with the plugin as a generator, example:

```shell
kubectl apply -f applicationset-demo.yaml
```
