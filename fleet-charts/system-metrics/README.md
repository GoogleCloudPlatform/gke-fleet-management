# System Metrics OpenTelemetry Collector Helm Chart

This Helm chart deploys the Google-Built OpenTelemetry Collector (`otelcol-google`) as a `DaemonSet` in `kube-system` to collect host system metrics and Kubelet node/pod/container/volume metrics and export them to Google Cloud Managed Service for Prometheus (GMP) using Fleet Workload Identity Federation (WIF).

## Features

- **Keyless Authentication (Fleet WIF)**: Authenticates to Google Cloud Monitoring / Managed Service for Prometheus using a projected Kubernetes Service Account token (`<PROJECT_ID>.svc.id.goog`) and an `external_account` Application Default Credentials configuration—no service account keys required.
- **Optional Service Account Impersonation**: Supports impersonating a Google Cloud Service Account (`serviceAccountEmail`) when required by VPC Service Controls (VPC-SC).
- **Read-Only Kubernetes RBAC**: Grants only `get`, `list`, and `watch` permissions on `pods`, `namespaces`, `nodes`, `nodes/stats`, `nodes/proxy`, and `replicasets` for local Kubelet metric scraping and Kubernetes metadata enrichment.
- **Host & Kubelet Metric Collection**:
  - Host metrics (`cpu`, `memory`, `load`, `paging`, `disk`, `filesystem`, `network`, `processes`) via the `hostmetrics` receiver reading `/hostfs`.
  - Kubelet metrics (`node`, `pod`, `container`, `volume`) via the `kubeletstats` receiver scraping the local node Kubelet endpoint (`https://${NODE_IP}:10250`).

## Prerequisites

1. Register the cluster with a Fleet membership that has Workload Identity enabled:

   ```bash
   gcloud container fleet memberships create <MEMBERSHIP_NAME> \
     --project=<PROJECT_ID> \
     --location=<LOCATION> \
     --enable-workload-identity \
     --has-private-issuer \
     --membership-type=READONLY
   ```

2. Grant `roles/monitoring.metricWriter` to the Fleet Workload Identity principal (or to the impersonated Google Service Account if using `serviceAccountEmail`). Replace `<NAMESPACE>` with the chart's `namespace` value (default: `kube-system`):

   ```bash
   gcloud projects add-iam-policy-binding <PROJECT_ID> \
     --member="principal://iam.googleapis.com/projects/<PROJECT_NUMBER>/locations/global/workloadIdentityPools/<PROJECT_ID>.svc.id.goog/subject/ns/<NAMESPACE>/sa/system-metrics-otel-collector" \
     --role="roles/monitoring.metricWriter"
   ```

## Installation

Install the chart with your Fleet project and membership details:

```bash
helm install system-metrics ./fleet-charts/system-metrics \
  --set projectId=<PROJECT_ID> \
  --set membershipName=<MEMBERSHIP_NAME> \
  --set location=<GCP_REGION_OR_ZONE>
```

On GKE clusters with Workload Identity enabled (`gke-metadata-server`), add `--set useProjectedKsaToken=false` to authenticate through the GKE metadata server instead of the projected token and credential file.

## Configuration Values

- `projectId` (required): GCP Fleet Project ID.
- `membershipName` (required): Fleet membership name.
- `location` (required): Cloud Monitoring location — must be a GCP region or zone (e.g., `"us-central1"`). `"global"` is not allowed by the `prometheus_target` Monitored Resource.
- `membershipLocation` (default: `"global"`): Fleet membership location (e.g., `"global"`, `"us-central1"`).
- `clusterName` (default: `""`): Cluster name label in Cloud Monitoring (defaults to `membershipName` when empty).
- `namespace` (default: `"kube-system"`): Kubernetes namespace for the collector resources.
- `serviceAccountEmail` (default: `""`): Optional Google Cloud Service Account email for Fleet WIF impersonation.
- `useProjectedKsaToken` (default: `true`): When `true`, authenticates with a projected KSA token and an `external_account` credential `Secret` (required for clusters without the GKE metadata server and for VPC-SC impersonation). When `false`, relies on the GKE metadata server and, if `serviceAccountEmail` is set, annotates the `ServiceAccount` with `iam.gke.io/gcp-service-account`.
- `collectionInterval` (default: `"60s"`): Scrape interval for `hostmetrics` and `kubeletstats` receivers.
- `image.repository` (default: `"us-docker.pkg.dev/cloud-ops-agents-artifacts/google-cloud-opentelemetry-collector/otelcol-google"`): Collector image repository.
- `image.tag` (default: `"0.156.1"`): Collector image tag.
- `resources`: CPU and memory requests/limits for the collector container.
- `tolerations` (default: `[{operator: "Exists"}]`): Pod tolerations for the `DaemonSet`.
