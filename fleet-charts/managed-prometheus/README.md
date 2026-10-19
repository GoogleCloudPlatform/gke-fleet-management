# Managed Service for Prometheus Helm Chart

This Helm chart deploys [Google Cloud Managed Service for Prometheus](https://cloud.google.com/stackdriver/docs/managed-prometheus) (GMP) managed collection: the GMP operator, collector `DaemonSet`, rule-evaluator, and Alertmanager. It's meant for clusters outside GKE, such as attached or on-prem clusters, and authenticates with Fleet Workload Identity Federation (WIF).

It's based on the upstream [prometheus-engine v0.17.0](https://github.com/GoogleCloudPlatform/prometheus-engine/tree/v0.17.0/charts/operator) chart. With `useProjectedKsaToken=false` and no tolerations, it renders the same resources as upstream's `manifests/operator.yaml`, apart from the operator's `--project-id`, `--cluster`, and `--location` flags. Upstream's charts are templating sources for its manifests and aren't published for direct installation, so this chart packages them for Fleet use.

## Features

- **Keyless Authentication (Fleet WIF)**: The collector and rule-evaluator authenticate with a projected Kubernetes Service Account token (`<PROJECT_ID>.svc.id.goog`) and an `external_account` credential, referenced from the `OperatorConfig`. No service account keys are required.
- **Optional Service Account Impersonation**: Supports impersonating a Google Cloud Service Account (`serviceAccountEmail`) when required by VPC Service Controls (VPC-SC).
- **Managed Collection**: Scrape targets are configured with GMP's `PodMonitoring`, `ClusterPodMonitoring`, and `ClusterNodeMonitoring` resources, and rules with `Rules`, `ClusterRules`, and `GlobalRules`. Metrics are exported to Cloud Monitoring as the `prometheus_target` Monitored Resource.

## Prerequisites

1. Kubernetes 1.30 or later (the chart uses `admissionregistration.k8s.io/v1` `ValidatingAdmissionPolicy`).

2. Register the cluster with a Fleet membership that has Workload Identity enabled:

   ```bash
   gcloud container fleet memberships create <MEMBERSHIP_NAME> \
     --project=<PROJECT_ID> \
     --location=<LOCATION> \
     --enable-workload-identity \
     --has-private-issuer \
     --membership-type=READONLY
   ```

3. Grant `roles/monitoring.metricWriter` (export) and `roles/monitoring.viewer` (rule evaluation queries) to the Fleet Workload Identity principal, or to the impersonated Google Service Account if using `serviceAccountEmail`. The collector and rule-evaluator both run as the `collector` Kubernetes service account in `gmp-system`:

   ```bash
   for role in roles/monitoring.metricWriter roles/monitoring.viewer; do
     gcloud projects add-iam-policy-binding <PROJECT_ID> \
       --member="principal://iam.googleapis.com/projects/<PROJECT_NUMBER>/locations/global/workloadIdentityPools/<PROJECT_ID>.svc.id.goog/subject/ns/gmp-system/sa/collector" \
       --role="$role"
   done
   ```

> [!IMPORTANT]
> Don't install this chart on GKE clusters with GKE's built-in managed collection enabled (the default on recent GKE versions). GKE already runs these components in `gmp-system`, and the two would conflict.

## Installation

Install the chart with your Fleet project and membership details:

```bash
helm install managed-prometheus ./fleet-charts/managed-prometheus \
  --set projectId=<PROJECT_ID> \
  --set membershipName=<MEMBERSHIP_NAME> \
  --set location=<GCP_REGION_OR_ZONE>
```

Then configure what to scrape, for example:

```yaml
apiVersion: monitoring.googleapis.com/v1
kind: PodMonitoring
metadata:
  name: my-app
  namespace: my-namespace
spec:
  selector:
    matchLabels:
      app: my-app
  endpoints:
  - port: metrics
    interval: 30s
```

> [!NOTE]
> Helm installs the GMP CRDs from `crds/` on the first install only, and never upgrades or deletes them. When a new chart version updates the CRDs, apply them with `kubectl apply --server-side -f fleet-charts/managed-prometheus/crds/` before upgrading.

## Configuration Values

- `projectId` (required): GCP Fleet Project ID.
- `membershipName` (required): Fleet membership name.
- `location` (required): Cloud Monitoring location. It must be a GCP region or zone (e.g., `"us-central1"`); `"global"` isn't allowed by the `prometheus_target` Monitored Resource.
- `membershipLocation` (default: `"global"`): Fleet membership location.
- `clusterName` (default: `""`): `cluster` label in Cloud Monitoring (defaults to `membershipName` when empty).
- `serviceAccountEmail` (default: `""`): Optional Google Cloud Service Account email for Fleet WIF impersonation.
- `useProjectedKsaToken` (default: `true`): When `true`, the collector and rule-evaluator authenticate with a projected KSA token and an `external_account` credential `Secret` (`gmp-wif-credentials` in `gmp-public`). When `false`, they rely on the GKE metadata server and, if `serviceAccountEmail` is set, the `collector` ServiceAccount is annotated with `iam.gke.io/gcp-service-account`. Only use `false` on GKE clusters with managed collection disabled.
- `tolerations` (default: `[]`): Pod tolerations appended to the `gmp-operator`, `rule-evaluator`, and `alertmanager` pods. The collector `DaemonSet` already tolerates all taints.
- `images`, `resources`, `namespace`, and the other values follow upstream's [`values.global.yaml`](https://github.com/GoogleCloudPlatform/prometheus-engine/blob/v0.17.0/charts/values.global.yaml).
