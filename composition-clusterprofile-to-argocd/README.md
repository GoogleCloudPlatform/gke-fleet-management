# Translating ClusterProfile into ArgoCD with compositions

Translate ClusterProfile coming from GKE Fleet into ArgoCD Cluster secrets automatically using compositions.
Composition makes it super simple to translate ClusterProfiles (the shared facade) into whatever argocd (the multicluster controller) needs to work on a bunch of clusters. This example focuses on working with GKE but any ClusterManager would work (but maybe more complicated on the auth side!)

## Assumptions

This tool assumes the Clusterprofiles are setup with a certain configuration:
* Uses a GKE cluster endpoint (the secret configuration is hardcoded to GCP auth); recommended via GKE Fleet Connect Gateway to limit configuration requirements.
* Has an annotation
* (INSECURE, kept for simplification) ClusterProfiles from any namespace are translated into argocd namespace.
* Management cluster (where argocd and ClusterProfiles are installed) uses GKE WI


## Install:

```
# Get a kubeconfig
gcloud container fleet memberships get-credentials $MEMBERSHIP_NAME

# Install composition controller
MANIFEST_URL=https://raw.githubusercontent.com/GoogleCloudPlatform/k8s-config-connector/master/experiments/compositions/composition/release/manifest.yaml
kubectl apply -f ${MANIFEST_URL}

# Install our translation composition
kubectl apply -f composition.yaml
```

### Configure ArgoCD

#### Argo permissions
```
# Follow your favorite argocd installation, but can be:
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

# Then give permission to argocd on all the workload clusters:
PROJECT_ID=$(gcloud config get-value project)

gcloud iam service-accounts create argocd \
  --description="argocd on " \
  --display-name="ArgoCD"

gcloud iam service-accounts add-iam-policy-binding argocd@${PROJECT_ID}.iam.gserviceaccount.com \
    --role roles/iam.workloadIdentityUser \
    --member "serviceAccount:${PROJECT_ID}.svc.id.goog[argocd/argocd-application-controller]"
gcloud iam service-accounts add-iam-policy-binding argocd@${PROJECT_ID}.iam.gserviceaccount.com \
    --role roles/iam.workloadIdentityUser \
    --member "serviceAccount:${PROJECT_ID}.svc.id.goog[argocd/argocd-applicationset-controller]"
gcloud iam service-accounts add-iam-policy-binding argocd@${PROJECT_ID}.iam.gserviceaccount.com \
    --role roles/iam.workloadIdentityUser \
    --member "serviceAccount:${PROJECT_ID}.svc.id.goog[argocd/argocd-server]"

# Argocd service accounts must be configured to use the newly-created GSA (for workload identity)
gcloud container fleet memberships get-credentials $MEMBERSHIP_NAME
kubectl annotate serviceaccount -n argocd argocd-application-controller \
  "iam.gke.io/gcp-service-account"="argocd@${PROJECT_ID}.iam.gserviceaccount.com"
kubectl annotate serviceaccount -n argocd argocd-applicationset-controller \
  "iam.gke.io/gcp-service-account"="argocd@${PROJECT_ID}.iam.gserviceaccount.com"
kubectl annotate serviceaccount -n argocd argocd-server \
  "iam.gke.io/gcp-service-account"="argocd@${PROJECT_ID}.iam.gserviceaccount.com"

# Once for the entire Fleet, argocd must be given connect gateway usage.
gcloud projects add-iam-policy-binding ${PROJECT_ID} \
  --member="serviceAccount:argocd@${PROJECT_ID}.iam.gserviceaccount.com" \
  --role="roles/gkehub.gatewayEditor" \
  --condition=None

# For each cluster project, argocd must be given access inside the cluster (so it can use kubernetes resources).
# This can be replaced by RBAC.
gcloud projects add-iam-policy-binding ${PROJECT_ID} \
  --member="serviceAccount:argocd@${PROJECT_ID}.iam.gserviceaccount.com" \
  --role="roles/container.developer" \
  --condition=None
```
