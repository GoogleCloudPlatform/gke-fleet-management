# Cloud Logging OpenTelemetry Collector Helm Chart

This Helm chart deploys the Google-Built OpenTelemetry Collector (`otelcol-google`) as a `DaemonSet` in `kube-system` to collect workload container logs and host system logs and export them to Google Cloud Logging using Fleet Workload Identity Federation (WIF).

## Features

- **Keyless Authentication (Fleet WIF)**: Authenticates to Google Cloud Logging using a projected Kubernetes Service Account token (`<PROJECT_ID>.svc.id.goog`) and an `external_account` Application Default Credentials configuration—no service account keys required.
- **Optional Service Account Impersonation**: Supports impersonating a Google Cloud Service Account (`serviceAccountEmail`) when required by VPC Service Controls (VPC-SC).
- **Read-Only Kubernetes RBAC**: Grants only `get`, `list`, and `watch` permissions on `pods`, `namespaces`, `nodes`, and `replicasets` for Kubernetes metadata enrichment (plus `events` when event collection is enabled).
- **Workload & System Log Collection**:
  - Container logs from `/var/log/pods/*/*/*.log` mapped to the `k8s_container` Monitored Resource.
  - Host system logs (`/var/log/syslog`, `/var/log/messages`, `/var/log/kern.log`) mapped to the `k8s_node` Monitored Resource.
  - Host system logs from journald (`journald.units`, default kubelet, containerd, and docker, plus kernel messages) mapped to the `k8s_node` Monitored Resource, one log per unit (`kubelet`, `containerd`, `kernel`). A Fluent Bit sidecar reads the journal natively and forwards entries to the collector over localhost.
  - Kubernetes events written to the `events` log by a single-replica `Deployment` (`events.enabled`), with the involved object as `k8s.object.*` labels. It reuses the collector's ServiceAccount, so no extra IAM binding is needed.

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

2. Grant `roles/logging.logWriter` to the Fleet Workload Identity principal (or to the impersonated Google Service Account if using `serviceAccountEmail`). Replace `<NAMESPACE>` with the chart's `namespace` value (default: `kube-system`):

   ```bash
   gcloud projects add-iam-policy-binding <PROJECT_ID> \
     --member="principal://iam.googleapis.com/projects/<PROJECT_NUMBER>/locations/global/workloadIdentityPools/<PROJECT_ID>.svc.id.goog/subject/ns/<NAMESPACE>/sa/cloud-logging-otel-collector" \
     --role="roles/logging.logWriter"
   ```

## Installation

Install the chart with your Fleet project and membership details:

```bash
helm install cloud-logging ./fleet-charts/cloud-logging \
  --set projectId=<PROJECT_ID> \
  --set membershipName=<MEMBERSHIP_NAME>
```

On GKE clusters with Workload Identity enabled (`gke-metadata-server`), add `--set useProjectedKsaToken=false` to authenticate through the GKE metadata server instead of the projected token and credential file.

## Configuration Values

- `projectId` (required): GCP Fleet Project ID.
- `membershipName` (required): Fleet membership name.
- `location` (default: `"global"`): Cloud Logging Monitored Resource location (e.g., `"global"`, `"us-central1"`).
- `membershipLocation` (default: `"global"`): Fleet membership location (e.g., `"global"`, `"us-central1"`).
- `clusterName` (default: `""`): Cluster name label in Cloud Logging (defaults to `membershipName` when empty).
- `namespace` (default: `"kube-system"`): Kubernetes namespace for the collector resources.
- `serviceAccountEmail` (default: `""`): Optional Google Cloud Service Account email for Fleet WIF impersonation.
- `useProjectedKsaToken` (default: `true`): When `true`, authenticates with a projected KSA token and an `external_account` credential `Secret` (required for clusters without the GKE metadata server and for VPC-SC impersonation). When `false`, relies on the GKE metadata server and, if `serviceAccountEmail` is set, annotates the `ServiceAccount` with `iam.gke.io/gcp-service-account`.
- `image.repository` (default: `"us-docker.pkg.dev/cloud-ops-agents-artifacts/google-cloud-opentelemetry-collector/otelcol-google"`): Collector image repository.
- `image.tag` (default: `"0.156.1"`): Collector image tag.
- `systemLogs.include` (default: `["/var/log/syslog", "/var/log/messages", "/var/log/kern.log"]`): Host system log file paths to tail. On nodes running rsyslog (common on Debian and Ubuntu server installs), which copies the journal into these files, journald entries are exported twice; remove the files or disable `journald`.
- `resources`: CPU and memory requests/limits for the collector container.
- `tolerations` (default: `[{operator: "Exists"}]`): Pod tolerations for the `DaemonSet`.
- `journald.enabled` (default: `true`): Collect host logs from journald. The sidecar mounts the host's `/etc/machine-id`, so set this to `false` on nodes without systemd, or the collector pod won't start.
- `journald.units` (default: `[kubelet.service, containerd.service, docker.service]`): systemd units to collect, including the type suffix (for example `kubelet.service`, not `kubelet`). Other distributions use other unit names, for example `crio.service`, `k3s.service`, or `rke2-server.service`.
- `journald.kernel` (default: `true`): Also collect kernel messages.
- `journald.image.repository` / `journald.image.tag` (default: `cr.fluentbit.io/fluent/fluent-bit:5.1.3`): journald reader image. Mirror it for air-gapped clusters.
- `journald.resources`: CPU and memory requests/limits for the journald reader. Unsent entries are buffered on the host (up to 256 MiB, under `/var/lib/otelcol/file_storage/`) and sent after restarts. The read position (`journald.db`) and buffer (`journald-buffer/`) stay on the host if journald is later disabled; delete them to reclaim the space.
- `events.enabled` (default: `true`): Collect Kubernetes events into the `events` log.
- `events.resources`: CPU and memory requests/limits for the events collector.
- `events.nodeSelector` (default: `{}`): Node selector for the events `Deployment`.
- `events.tolerations` (default: `[]`): Pod tolerations for the events `Deployment`. Set these if every node in the cluster is tainted.

> [!NOTE]
> The events receiver only exports events created after it starts. During a rollout, the new events pod starts
> before the old one stops, so events from the brief overlap may be exported twice. If the pod crashes or its
> node fails, events that occur before the replacement starts aren't collected. The events `Deployment` only
> rolls out when its own settings, collector config, or credentials change, not for `DaemonSet`-only values
> such as `resources`, `tolerations`, `systemLogs`, and `journald`, or for chart upgrades that don't change it.
