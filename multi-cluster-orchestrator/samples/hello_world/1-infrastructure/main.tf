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

# [START gke_mco_hello_world_1_infrastructure]
locals {
  hub_cluster_location       = "us-central1"
  workload_cluster_locations = ["us-west1", "us-east1", "europe-west4"]

  argocd_version = "7.8.28" # Choose from https://github.com/argoproj/argo-helm/releases
}

data "google_project" "default" {}

### Enable Services
resource "google_project_service" "default" {
  for_each = toset([
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

# Service Account of the MCS Importer
# https://cloud.google.com/kubernetes-engine/docs/how-to/multi-cluster-services#enabling
resource "google_project_iam_member" "gke-mcs-importer" {
  project = data.google_project.default.project_id

  role   = "roles/compute.networkViewer"
  member = "principal://iam.googleapis.com/projects/${data.google_project.default.number}/locations/global/workloadIdentityPools/${data.google_project.default.project_id}.svc.id.goog/subject/ns/gke-mcs/sa/gke-mcs-importer"

  # Requires Identity Pool
  depends_on = [google_container_cluster.hub]
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

  # Set `deletion_protection` to `true` will ensure that one cannot
  # accidentally delete this instance by use of Terraform.
  deletion_protection = false

  depends_on = [google_project_service.default]
}

# Apply label to membership without importing the membership
module "gcloud" {
  source  = "terraform-google-modules/gcloud/google"
  version = "~> 3.5"

  platform = "linux"

  create_cmd_entrypoint = "gcloud"
  create_cmd_body       = "container fleet memberships update ${google_container_cluster.hub.name} --update-labels=\"fleet-clusterinventory-management-cluster=true\" --location ${google_container_cluster.hub.location} --project ${google_container_cluster.hub.project}"
}

## Workload Clusters
resource "google_container_cluster" "clusters" {
  for_each = toset(local.workload_cluster_locations)

  name             = "mco-cluster"
  location         = each.value
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

  # Set `deletion_protection` to `true` will ensure that one cannot
  # accidentally delete this instance by use of Terraform.
  deletion_protection = false

  depends_on = [google_project_service.default]
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

  depends_on = [google_project_service.default]
}

### Helm Releases
data "google_client_config" "default" {}

provider "helm" {
  kubernetes {
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
  value       = helm_release.argocd.metadata[0].notes
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

# NOTE: Helm does not support upgrading CRDs at this time.
# https://helm.sh/docs/chart_best_practices/custom_resource_definitions/#some-caveats-and-explanations
resource "helm_release" "orchestrator" {
  name       = "orchestrator"
  repository = "https://googlecloudplatform.github.io/gke-fleet-management"
  chart      = "orchestrator"
  version    = "0.0.4"

  lint = true

  depends_on = [helm_release.argocd]
}

resource "helm_release" "argocd-clusterprofile-syncer" {
  name       = "argocd-clusterprofile-syncer"
  repository = "https://googlecloudplatform.github.io/gke-fleet-management"
  chart      = "argocd-clusterprofile-syncer"
  version    = "0.0.1"

  # Deploy into the same namespace as ArgoCD
  namespace = helm_release.argocd.namespace

  lint = true

  depends_on = [helm_release.orchestrator]
}

resource "helm_release" "argocd-mco-plugin" {
  name       = "argocd-mco-plugin"
  repository = "https://googlecloudplatform.github.io/gke-fleet-management"
  chart      = "argocd-mco-plugin"
  version    = "0.0.1"

  # Deploy into the same namespace as ArgoCD
  namespace = helm_release.argocd.namespace

  lint = true

  depends_on = [helm_release.orchestrator]
}
# [END gke_mco_hello_world_1_infrastructure]
