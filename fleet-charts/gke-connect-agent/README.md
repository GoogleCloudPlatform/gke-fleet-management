# GKE Connect Agent Helm Chart (Minimal / Read-Only RBAC)

This Helm chart deploys the Connect Agent with minimal, namespace-scoped RBAC permissions and Workload Identity Federation (WIF). It omits cluster-wide impersonation, Feature Authorizer (`cluster-admin`), and Anthos Identity Service (`ClusterRole`) bindings so that external services cannot mutate cluster resources.

The only write permissions given to Connect Agent are create/update event objects in its namespace, to create status events such as "SessionEstablished."

## Prerequisites

Create the Fleet membership with Workload Identity enabled:

```bash
gcloud container fleet memberships create <MEMBERSHIP_NAME> \
  --project=<PROJECT_ID> \
  --location=<LOCATION> \
  --enable-workload-identity \
  --has-private-issuer \
  --membership-type=READONLY
```

## Installation

Install the chart with your Fleet project and membership details:

```bash
helm install gke-connect-agent ./fleet-charts/gke-connect-agent \
  --set projectId=<PROJECT_ID> \
  --set projectNumber=<PROJECT_NUMBER> \
  --set membershipName=<MEMBERSHIP_NAME> \
  --set location=<LOCATION>
```

## Configuration Values

- `projectId` (required): GCP Fleet Project ID.
- `projectNumber` (required): GCP Fleet Project Number.
- `membershipName` (required): Fleet membership name.
- `location` (default: `"global"`): Fleet membership location (e.g., `"global"`, `"us-central1"`).
- `namespace` (default: `"gke-connect"`): Kubernetes namespace for the Connect Agent resources.
- `replicas` (default: `2`): Number of Connect Agent replicas.
- `gkeConnectApiEndpoint` (default: `"gkeconnect.googleapis.com:443"`): GKE Connect API endpoint.
- `image.repository` (default: `"gcr.io/gkeconnect/gkeconnect-gce"`): Connect Agent image repository.
- `image.tag` (default: `"20261001-03-00"`): Connect Agent image tag.
- `proxy.httpProxy` (default: `""`): Optional HTTP/HTTPS proxy URL.
- `tolerations` (default: `node-role.kubernetes.io/control-plane:NoSchedule` with `operator: Exists`): Pod tolerations for the Connect Agent deployment.

Both Project ID and number are required. ID is needed for Workload Identity, number for Connect.
