# ArgoCD Fleet Plugin

## Overview
An example application that Fleet customers can copy and deploy as an [ArgoCD plugin generator](https://argo-cd.readthedocs.io/en/stable/operator-manual/applicationset/Generators-Plugin/#plugin-generator).

*   Disclaimer: This is not an official Google product
*   Created by: GKE Fleet Teams

### Prerequisites
- Clone this repo.
- Setup fleet (membership/teams).
- Enable Connect-gateway on all your clusters.

### Build
#### Create an artifacts repository to store the container image for the plugin.

Note: This is a one-time setup.

```shell
gcloud artifacts repositories create argocd-fleet-sync \
    --project={$PROJECT_ID} \
    --repository-format=docker \
    --location={$LOCATION} \
    --description="Docker repository for argocd fleet plugin"
```

#### Build and upload image using Cloud Build.

```shell
gcloud builds submit --region=us-central1 --config=cloudbuild.yaml
```

Navigate to Google Cloud UI to confirm the image has been successfully uploaded.

### Deploy

This application would be deployed on your argocd control plane cluster.

#### Setup GKE Workload identity federation

Get credentials for your cluster:

```shell
gcloud container clusters get-credentials CLUSTER_NAME \
    --location=LOCATION
```
This example uses [GKE workload identity](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity) federation to authenticate to GCP APIs. 
* Enable workload identity federation on the control clusters and its nodepools. [guide](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity#enable-existing-cluster)

* Create iam principle which the pod will be running as, and give it nessary permissions.

    ```shell
    gcloud projects add-iam-policy-binding projects/PROJECT_ID \
    --role=roles/gkehub.admin \
    --member=principal://iam.googleapis.com/projects/PROJECT_NUMBER/locations/global/workloadIdentityPools/PROJECT_ID.svc.id.goog/subject/ns/argocd/sa/argocd-fleet-sync \
    ```

* Give your node service account necessary permissions to fetch image in artifacts repository. 

    ```shell
    gcloud projects add-iam-policy-binding projects/PROJECT_ID \
    --role="roles/artifactregistry.reader" \
    --member=principal:PROJECT-NUMBER-compute@developer.gserviceaccount.com \
    ```

#### Run fleet plugin on the control cluster.

Replace `container.image` in `argocd-fleet-sync-install.yaml` with the actual
path to image, and run:

```shell
kubectl apply -f argocd-fleet-sync-install.yaml -n argocd
```

Now we are ready to use the fleet argocd plugin in the ApplicationSet. Modify your applicationSet to adopt the plugin:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: webserver-applicationset
  namespace: argocd
spec:
    generators:
    - plugin:
        configMapRef:
          name: fleet-plugin
        input:
          parameters:
            fleetProjectNumber: {PROJECT_NUM}
            scopeID: {SCOPE_ID}
        requeueAfterSeconds: 10
    syncPolicy:
      applicationsSync: sync
      preserveResourcesOnDeletion: false
    template:
      metadata:
        name: '{{name}}-webserver'
      spec:
        destination:
          namespace: webserver
          server: '{{server}}'
        project: default
        source:
          path: {PATH}
          repoURL: {URL}
          targetRevision: HEAD
    syncPolicy:
      # The controller will delete Applications when the ApplicationSet is deleted.
      preserveResourcesOnDeletion: false
```
