# GKE Fleet Management Helm chart repository

This site hosts the [Helm](https://helm.sh/) chart repository for the [`GoogleCloudPlatform/gke-fleet-management`](https://github.com/GoogleCloudPlatform/gke-fleet-management) repository, which provides sample Helm charts for managing clusters with [GKE fleets](https://cloud.google.com/kubernetes-engine/fleet-management/docs).

## Add the Helm repository

To add this Helm repository and fetch the latest chart index, run the following commands:

```bash
helm repo add gke-fleet https://googlecloudplatform.github.io/gke-fleet-management/
helm repo update
```

To list all available charts and versions in the repository, run the following command:

```bash
helm search repo gke-fleet
```

To install a chart from the repository, run the following command:

```bash
helm install RELEASE_NAME gke-fleet/CHART_NAME
```

Replace the following:

- `RELEASE_NAME`: the name that you choose for the Helm release.
- `CHART_NAME`: the name of the chart to install.

## Available charts

For the complete list of available charts, versions, and chart documentation, see the following resources:

- [`fleet-charts` directory on GitHub](https://github.com/GoogleCloudPlatform/gke-fleet-management/tree/main/fleet-charts): chart source files, configuration values, and usage documentation.
- [Helm repository index (`index.yaml`)](./index.yaml): the generated Helm chart index served from this branch.

## Get help

Use the [Issues page](https://github.com/GoogleCloudPlatform/gke-fleet-management/issues) to provide feedback or submit a bug report.

## Disclaimer

This repository is not an officially supported Google product. The code and charts in this repository are for demonstrative purposes only.

## References

- [`GoogleCloudPlatform/gke-fleet-management` repository](https://github.com/GoogleCloudPlatform/gke-fleet-management)
- [Fleet Management overview](https://cloud.google.com/kubernetes-engine/fleet-management/docs)
- [Fleet concepts](https://cloud.google.com/kubernetes-engine/fleet-management/docs/fleet-concepts)
