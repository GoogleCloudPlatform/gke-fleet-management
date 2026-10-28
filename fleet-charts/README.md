# Fleet Charts

Sample [Helm](https://helm.sh/) charts for use with [fleets](https://cloud.google.com/kubernetes-engine/fleet-management/docs). Released chart versions are published to a Helm repository:

```bash
helm repo add gke-fleet-management https://googlecloudplatform.github.io/gke-fleet-management
helm search repo gke-fleet-management
```

## Charts for non-GKE clusters with Fleet Workload Identity

These charts connect and observe non-GKE clusters, such as on-prem or bare-metal clusters, that are registered to a Fleet with Workload Identity. They authenticate to Google Cloud with Workload Identity Federation instead of service account keys, and use read-only Kubernetes RBAC.

| Chart | Description |
| --- | --- |
| [`gke-connect-agent`](./gke-connect-agent) | Connect Agent with minimal, read-only RBAC. |
| [`cloud-logging`](./cloud-logging) | Container, host, and journald logs, plus Kubernetes events, exported to Cloud Logging. |
| [`system-metrics`](./system-metrics) | Host, kubelet, and cluster-state metrics, exported to Managed Service for Prometheus. |

### Managed Service for Prometheus

There's no chart here for [Managed Service for Prometheus](https://cloud.google.com/stackdriver/docs/managed-prometheus) (GMP) managed collection, which scrapes your applications' metrics through `PodMonitoring` resources. GMP's maintainers support installing it from the released GMP manifests rather than from Helm charts, so use the [`examples/fleet-workload-identity`](https://github.com/GoogleCloudPlatform/prometheus-engine/tree/main/examples/fleet-workload-identity) Kustomize overlay in prometheus-engine. It configures the GMP operator, collector, and rule-evaluator for Fleet Workload Identity on non-GKE clusters.

GMP's collector runs as its own Kubernetes service account (`gmp-system/collector`), separate from `system-metrics`, so it needs its own IAM grants. When using both, leave GMP's kubelet scraping off (the default) to avoid collecting kubelet metrics twice.

## Other charts

| Chart | Description |
| --- | --- |
| [`argocd-clusterprofile-syncer`](./argocd-clusterprofile-syncer) | Argo CD ClusterProfile Syncer. |
| [`argocd-mco-plugin`](./argocd-mco-plugin) | Argo CD plugin for Multi-Cluster Orchestrator. |
| [`orchestrator`](./orchestrator) | Multi-Cluster Orchestrator. |
| [`gemma-server`](./gemma-server) | Serves Gemma 3 open models on GPUs with vLLM. |
| [`hello-world`](./hello-world) | Sample Deployment with a HorizontalPodAutoscaler, Service, and ServiceExport. |
