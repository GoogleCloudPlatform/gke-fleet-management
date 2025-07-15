/**
* Copyright 2025 Google LLC
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*
*      http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
*/

# [START gke_mco_gemma_vllm_1_infrastructure]
locals {
  hub_cluster_location       = "us-central1"
  workload_cluster_locations = ["us-west1", "us-east1", "europe-west4"]

  argocd_version        = "7.8.28" # Choose from https://github.com/argoproj/argo-helm/releases?q=argo-cd
  cm_sd_adapter_version = "0.16.1" # Choose from https://github.com/GoogleCloudPlatform/k8s-stackdriver/releases?q=cm-sd-adapter
}

data "google_project" "default" {}

### Enable Services
resource "google_project_service" "default" {
  for_each = toset([
    "cloudresourcemanager.googleapis.com",
    "compute.googleapis.com",
    "container.googleapis.com",
    "gkehub.googleapis.com",
    "connectgateway.googleapis.com",
    "monitoring.googleapis.com",
    "trafficdirector.googleapis.com",
    "multiclusteringress.googleapis.com",
    "multiclusterservicediscovery.googleapis.com"
  ])

  service            = each.value
  disable_on_destroy = false
}

### Cluster Service Accounts
resource "google_service_account" "clusters" {
  for_each = toset([
    "hub",
    "worker"
  ])

  project      = data.google_project.default.project_id
  account_id   = "service-account-${each.key}-cluster"
  display_name = "Service Account for ${each.key} cluster"
}

# Cluster Service Account Permissions
resource "google_project_iam_member" "clusters" {
  for_each = {
    for o in distinct(flatten([
      for sa in google_service_account.clusters :
      [
        for role in [
          "roles/container.defaultNodeServiceAccount",
          "roles/monitoring.metricWriter",
          # This can be removed if nodes don't need to pull images from local Artifact Registry
          "roles/artifactregistry.reader",
          # For image streaming
          "roles/serviceusage.serviceUsageConsumer"
        ] :
        {
          "email" : sa.email,
          "role" : role,
        }
      ]
    ])) :
    "${o.email}/${o.role}" => o
  }

  project = data.google_project.default.project_id
  role    = each.value.role
  member  = "serviceAccount:${each.value.email}"
}

# Workload Identity Permissions

# Service Account of the Orchestrator Controller Manager
resource "google_project_iam_member" "hub-wi-mco-monitoring" {
  project = data.google_project.default.project_id

  role   = "roles/monitoring.viewer"
  member = "principal://iam.googleapis.com/projects/${data.google_project.default.number}/locations/global/workloadIdentityPools/${data.google_project.default.project_id}.svc.id.goog/subject/ns/orchestra-system/sa/orchestra-controller-manager"

  # Requires Identity Pool
  depends_on = [google_container_cluster.hub]
}

# Service Account of the Custom Metrics Stackdriver Adapter
resource "google_project_iam_member" "hub-wi-custom-metrics" {
  project = data.google_project.default.project_id

  role   = "roles/monitoring.viewer"
  member = "principal://iam.googleapis.com/projects/${data.google_project.default.number}/locations/global/workloadIdentityPools/${data.google_project.default.project_id}.svc.id.goog/subject/ns/custom-metrics/sa/custom-metrics-stackdriver-adapter"

  # Requires Identity Pool
  depends_on = [google_container_cluster.hub]
}

# For the MCS Importer
# https://cloud.google.com/kubernetes-engine/docs/how-to/multi-cluster-services#enabling
resource "google_project_iam_member" "gke-mcs-importer" {
  project = data.google_project.default.project_id

  role   = "roles/compute.networkViewer"
  member = "principal://iam.googleapis.com/projects/${data.google_project.default.number}/locations/global/workloadIdentityPools/${data.google_project.default.project_id}.svc.id.goog/subject/ns/gke-mcs/sa/gke-mcs-importer"

  # Requires Identity Pool
  depends_on = [google_container_cluster.hub]
}

### Networking
data "google_compute_network" "network" {
  name = "default"

  depends_on = [google_project_service.default]
}

resource "google_compute_subnetwork" "proxy" {
  name          = "proxy-subnetwork"
  ip_cidr_range = "10.3.0.0/22"
  region        = local.hub_cluster_location
  purpose       = "GLOBAL_MANAGED_PROXY"
  role          = "ACTIVE"
  network       = data.google_compute_network.network.id
}

### Hub Cluster
resource "google_container_cluster" "hub" {
  name             = "mco-hub"
  location         = local.hub_cluster_location
  enable_autopilot = true

  fleet {
    project = data.google_project.default.project_id
  }

  gateway_api_config {
    channel = "CHANNEL_STANDARD"
  }

  workload_identity_config {
    workload_pool = "${data.google_project.default.project_id}.svc.id.goog"
  }

  cluster_autoscaling {
    auto_provisioning_defaults {
      service_account = google_service_account.clusters["hub"].email
    }
  }

  release_channel {
    channel = "RAPID" # Greater than .1652000 for CRILB ephemeral addresses
  }

  resource_labels = {
    fleet-clusterinventory-management-cluster = true
  }

  # Set `deletion_protection` to `true` will ensure that one cannot
  # accidentally delete this instance by use of Terraform.
  deletion_protection = false

  depends_on = [google_project_service.default, google_project_iam_member.clusters["hub"]]
}

## Workload Clusters
resource "google_container_cluster" "clusters" {
  for_each = toset(local.workload_cluster_locations)

  name     = "mco-cluster"
  location = each.value

  fleet {
    project = data.google_project.default.project_id
  }

  gateway_api_config {
    channel = "CHANNEL_STANDARD"
  }

  workload_identity_config {
    workload_pool = "${data.google_project.default.project_id}.svc.id.goog"
  }

  initial_node_count = 1

  node_config {
    service_account = google_service_account.clusters["worker"].email
    gcfs_config {
      enabled = true
    }
  }

  monitoring_config {
    enable_components = ["SYSTEM_COMPONENTS", "APISERVER", "SCHEDULER", "CONTROLLER_MANAGER", "STORAGE", "HPA", "POD", "DAEMONSET", "DEPLOYMENT", "STATEFULSET", "KUBELET", "CADVISOR", "DCGM", "JOBSET"]
    managed_prometheus {
      enabled = true
      auto_monitoring_config {
        scope = "ALL"
      }
    }
  }

  cluster_autoscaling {
    autoscaling_profile = "OPTIMIZE_UTILIZATION"
  }

  # Set `deletion_protection` to `true` will ensure that one cannot
  # accidentally delete this instance by use of Terraform.
  deletion_protection = false

  depends_on = [google_project_service.default, google_project_iam_member.clusters["worker"]]
}

resource "google_container_node_pool" "gpu-node-pool" {
  for_each = google_container_cluster.clusters

  name    = "${each.value.name}-gpu-node-pool"
  cluster = each.value.id

  node_config {
    machine_type = "g2-standard-4"
    guest_accelerator {
      type  = "nvidia-l4"
      count = 1
      gpu_driver_installation_config {
        gpu_driver_version = "LATEST"
      }
    }
  }

  autoscaling {
    total_min_node_count = 1
    total_max_node_count = 3
  }
}

# Custom Metrics Stackdriver Adapter
# https://cloud.google.com/stackdriver/docs/managed-prometheus/hpa#stackdriver-adapter
module "custom-metrics-stackdriver-adapter" {
  source  = "terraform-google-modules/gcloud/google//modules/kubectl-fleet-wrapper"
  version = "~> 3.5"

  for_each = google_container_cluster.clusters

  membership_project_id = data.google_project.default.project_id
  membership_name       = each.value.fleet[0].membership_id
  membership_location   = each.value.fleet[0].membership_location

  kubectl_create_command  = "kubectl apply -f https://raw.githubusercontent.com/GoogleCloudPlatform/k8s-stackdriver/cm-sd-adapter-v${local.cm_sd_adapter_version}/custom-metrics-stackdriver-adapter/deploy/production/adapter_new_resource_model.yaml"
  kubectl_destroy_command = "kubectl delete -f https://raw.githubusercontent.com/GoogleCloudPlatform/k8s-stackdriver/cm-sd-adapter-v${local.cm_sd_adapter_version}/custom-metrics-stackdriver-adapter/deploy/production/adapter_new_resource_model.yaml"
}

### Multi-Cluster Gateway
resource "google_gke_hub_feature" "multiclusteringress" {
  name     = "multiclusteringress"
  location = "global"
  spec {
    multiclusteringress {
      config_membership = trimprefix(google_container_cluster.hub.fleet[0].membership, "//gkehub.googleapis.com/")
    }
  }

  depends_on = [google_project_service.default]
}

### Logging Metric
resource "google_logging_metric" "failed_scale_up_metric" {
  name   = "ClusterAutoscalerFailedScaleUp"
  filter = "jsonPayload.reason=FailedScaleUp AND jsonPayload.reportingComponent=cluster-autoscaler"
  label_extractors = {
    cluster        = "EXTRACT(resource.labels.cluster_name)"
    location       = "EXTRACT(resource.labels.location)"
    namespace_name = "EXTRACT(resource.labels.namespace_name)"
    pod_name       = "EXTRACT(resource.labels.pod_name)"
    project_id     = "EXTRACT(resource.labels.project_id)"
  }
  metric_descriptor {
    metric_kind  = "DELTA"
    display_name = "metric reflecting occurences of FailedScaleUp"
    value_type   = "INT64"
    labels {
      key = "namespace_name"
    }
    labels {
      key = "cluster"
    }
    labels {
      key = "pod_name"
    }
    labels {
      key = "project_id"
    }
    labels {
      key = "location"
    }
  }
}

### Manifests
data "google_client_config" "default" {}

provider "helm" {
  kubernetes = {
    host                   = "https://${google_container_cluster.hub.endpoint}"
    token                  = data.google_client_config.default.access_token
    cluster_ca_certificate = base64decode(google_container_cluster.hub.master_auth[0].cluster_ca_certificate)
  }
}

resource "helm_release" "argocd" {
  name             = "argocd"
  chart            = "https://github.com/argoproj/argo-helm/releases/download/argo-cd-${local.argocd_version}/argo-cd-${local.argocd_version}.tgz"
  namespace        = "argocd"
  create_namespace = true
  timeout          = 1200
}

# Provide guidance on accessing ArgoCD UI
output "argocd" {
  description = "ArgoCD Server UI Access"
  value       = helm_release.argocd.metadata.notes
}

# Service Account of the ArgoCD Application Controller
# Syncs the applications directly to the workload clusters
resource "google_project_iam_member" "hub-wi-argo-appsync" {
  project = data.google_project.default.project_id
  for_each = toset([
    "roles/gkehub.viewer",
    "roles/container.developer",
    "roles/gkehub.gatewayEditor"
  ])

  role   = each.value
  member = "principal://iam.googleapis.com/projects/${data.google_project.default.number}/locations/global/workloadIdentityPools/${data.google_project.default.project_id}.svc.id.goog/subject/ns/argocd/sa/argocd-application-controller"

  # Requires Identity Pool
  depends_on = [google_container_cluster.hub]
}

resource "helm_release" "orchestrator" {
  name       = "orchestrator"
  repository = "https://googlecloudplatform.github.io/gke-fleet-management"
  chart      = "orchestrator"
  version    = "0.1.0"

  lint = true

  depends_on = [helm_release.argocd]
}

resource "helm_release" "argocd-clusterprofile-syncer" {
  name       = "argocd-clusterprofile-syncer"
  repository = "https://googlecloudplatform.github.io/gke-fleet-management"
  chart      = "argocd-clusterprofile-syncer"
  version    = "0.1.0"

  # Deploy into the same namespace as ArgoCD
  namespace = helm_release.argocd.namespace

  lint = true

  depends_on = [helm_release.orchestrator]
}

resource "helm_release" "argocd-mco-plugin" {
  name       = "argocd-mco-plugin"
  repository = "https://googlecloudplatform.github.io/gke-fleet-management"
  chart      = "argocd-mco-plugin"
  version    = "0.1.0"

  # Deploy into the same namespace as ArgoCD
  namespace = helm_release.argocd.namespace

  lint = true

  depends_on = [helm_release.orchestrator]
}
# [END gke_mco_gemma_vllm_1_infrastructure]
