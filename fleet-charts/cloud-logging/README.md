# Cloud Logging OpenTelemetry Collector Helm Chart

This Helm chart deploys the Google-Built OpenTelemetry Collector (`otelcol-google`) as a `DaemonSet` in `kube-system` to collect workload container logs and host system logs and export them to Google Cloud Logging using Fleet Workload Identity Federation (WIF).

## Features

- **Keyless Authentication (Fleet WIF)**: Authenticates to Google Cloud Logging using a projected Kubernetes Service Account token (`<PROJECT_ID>.svc.id.goog`) and an `external_account` Application Default Credentials configuration—no service account keys required.
- **Optional Service Account Impersonation**: Supports impersonating a Google Cloud Service Account (`serviceAccountEmail`) when required by VPC Service Controls (VPC-SC).
- **Read-Only Kubernetes RBAC**: Grants only `get`, `list`, and `watch` permissions on `pods`, `namespaces`, `nodes`, and `replicasets` for Kubernetes metadata enrichment.
- **Workload & System Log Collection**:
  - Container logs from `/var/log/pods/*/*/*.log` mapped to the `k8s_container` Monitored Resource.
  - Host system logs (`/var/log/syslog`, `/var/log/messages`, `/var/log/kern.log`) mapped to the `k8s_node` Monitored Resource.

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

2. Grant `roles/logging.logWriter` to the Fleet Workload Identity principal (or to the impersonated Google Service Account if using `serviceAccountEmail`):

   ```bash
   gcloud projects add-iam-policy-binding <PROJECT_ID> \
     --member="principal://iam.googleapis.com/projects/<PROJECT_NUMBER>/locations/global/workloadIdentityPools/<PROJECT_ID>.svc.id.goog/subject/ns/kube-system/sa/cloud-logging-otel-collector" \
     --role="roles/logging.logWriter"
   ```

## Installation

Install the chart with your Fleet project and membership details:

```bash
helm install cloud-logging ./fleet-charts/cloud-logging \
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
- `clusterName` (default: `""`): Cluster name label in Cloud Logging (defaults to `membershipName` when empty).
- `namespace` (default: `"kube-system"`): Kubernetes namespace for the collector resources.
- `serviceAccountEmail` (default: `""`): Optional Google Cloud Service Account email for Fleet WIF impersonation.
- `image.repository` (default: `"us-docker.pkg.dev/cloud-ops-agents-artifacts/google-cloud-opentelemetry-collector/otelcol-google"`): Collector image repository.
- `image.tag` (default: `"0.156.1"`): Collector image tag.
- `systemLogs.include` (default: `["/var/log/syslog", "/var/log/messages", "/var/log/kern.log"]`): Host system log file paths to tail.
- `resources`: CPU and memory requests/limits for the collector container.
- `tolerations` (default: `[{operator: "Exists"}]`): Pod tolerations for the `DaemonSet`.
