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

locals {
  hub_cluster_location     = "us-central1"
  worker_cluster_locations = ["us-west1", "us-east1", "europe-west4"]
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
  ])

  service            = each.value
  disable_on_destroy = false
}

### Hub Cluster
resource "google_container_cluster" "hub" {
  name             = "multikueue-hub"
  location         = local.hub_cluster_location
  enable_autopilot = true

  fleet {
    project = data.google_project.default.project_id
  }

  workload_identity_config {
    workload_pool = "${data.google_project.default.project_id}.svc.id.goog"
  }

  resource_labels = {
    fleet-clusterinventory-management-cluster = true
    fleet-clusterinventory-namespace          = "kueue-system"
  }

  # Set `deletion_protection` to `true` will ensure that one cannot
  # accidentally delete this instance by use of Terraform.
  deletion_protection = false
}

resource "google_project_iam_member" "hub-wi-kueue-controller-manager" {
  project = data.google_project.default.project_id
  for_each = toset([
    "roles/container.developer",
    "roles/gkehub.gatewayEditor"
  ])

  role   = each.value
  member = "principal://iam.googleapis.com/projects/${data.google_project.default.number}/locations/global/workloadIdentityPools/${data.google_project.default.project_id}.svc.id.goog/subject/ns/kueue-system/sa/kueue-controller-manager"

  depends_on = [google_container_cluster.hub]
}

### Worker Clusters
resource "google_container_cluster" "clusters" {
  for_each = toset(local.worker_cluster_locations)

  name             = "multikueue-worker"
  location         = each.value
  enable_autopilot = true

  fleet {
    project = data.google_project.default.project_id
  }

  workload_identity_config {
    workload_pool = "${data.google_project.default.project_id}.svc.id.goog"
  }

  # Set `deletion_protection` to `true` will ensure that one cannot
  # accidentally delete this instance by use of Terraform.
  deletion_protection = false
}
