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

terraform {
  required_providers {
    google = { source = "hashicorp/google", version = ">= 5.0" }
    helm   = { source = "hashicorp/helm", version = ">= 2.10" }
  }
}

data "google_client_config" "default" {}

data "terraform_remote_state" "infra" {
  backend = "local"
  config = {
    path = "../1-infrastructure/terraform.tfstate"
  }
}

locals {
  kueue_version = "0.15.1"
  hub           = data.terraform_remote_state.infra.outputs.hub_cluster
  workers       = data.terraform_remote_state.infra.outputs.worker_clusters
}

provider "helm" {
  alias = "hub"
  kubernetes = {
    host                   = "https://${local.hub.endpoint}"
    token                  = data.google_client_config.default.access_token
    cluster_ca_certificate = base64decode(local.hub.ca_cert)
  }
}

module "kueue_hub" {
  source        = "./modules/kueue"
  kueue_version = local.kueue_version
  is_manager    = true
  providers     = { helm = helm.hub }
}

provider "helm" {
  alias = "worker0"
  kubernetes = {
    host                   = "https://${local.workers[0].endpoint}"
    token                  = data.google_client_config.default.access_token
    cluster_ca_certificate = base64decode(local.workers[0].ca_cert)
  }
}

module "kueue_worker_0" {
  source        = "./modules/kueue"
  kueue_version = local.kueue_version
  providers     = { helm = helm.worker0 }
}

provider "helm" {
  alias = "worker1"
  kubernetes = {
    host                   = "https://${local.workers[1].endpoint}"
    token                  = data.google_client_config.default.access_token
    cluster_ca_certificate = base64decode(local.workers[1].ca_cert)
  }
}
module "kueue_worker_1" {
  source        = "./modules/kueue"
  kueue_version = local.kueue_version
  providers     = { helm = helm.worker1 }
}

provider "helm" {
  alias = "worker2"
  kubernetes = {
    host                   = "https://${local.workers[2].endpoint}"
    token                  = data.google_client_config.default.access_token
    cluster_ca_certificate = base64decode(local.workers[2].ca_cert)
  }
}

module "kueue_worker_2" {
  source        = "./modules/kueue"
  kueue_version = local.kueue_version
  providers     = { helm = helm.worker2 }
}