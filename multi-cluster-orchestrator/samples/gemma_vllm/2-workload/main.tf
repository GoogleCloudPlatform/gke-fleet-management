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

# [START gke_mco_gemma_vllm_2_workload]
locals {
  hf_api_token = "REPLACE_WITH_YOUR_HF_API_TOKEN"
}

data "google_client_config" "default" {}

data "google_container_cluster" "hub" {
  name     = "mco-hub"
  location = "us-central1"
}

provider "helm" {
  kubernetes = {
    host                   = "https://${data.google_container_cluster.hub.endpoint}"
    token                  = data.google_client_config.default.access_token
    cluster_ca_certificate = base64decode(data.google_container_cluster.hub.master_auth[0].cluster_ca_certificate)
  }
}

resource "helm_release" "gemma-vllm-application" {
  name  = "gemma-vllm-application"
  chart = "./charts/gemma-vllm-application"

  namespace        = "gemma-server"
  create_namespace = true
  lint             = true

  set_sensitive = [
    {
      name  = "hf_api_token"
      value = local.hf_api_token
    }
  ]
  set = [
    {
      name  = "clusters_prefix"
      value = "mco-cluster"
    },
    {
      name  = "preferred_region"
      value = "europe-west4"
    }
  ]
}
# [END gke_mco_gemma_vllm_2_workload]
