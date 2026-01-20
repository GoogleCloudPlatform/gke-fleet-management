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

resource "helm_release" "kueue" {
  name             = "kueue"
  namespace        = "kueue-system"
  create_namespace = true
  repository       = "oci://registry.k8s.io/kueue/charts"
  chart            = "kueue"
  version          = var.kueue_version
  wait             = var.is_manager ? true : false

  values = var.is_manager ? [
    yamlencode({
      managerConfig = {
        controllerManagerConfigYaml = file("${path.module}/controller-manager-config.yaml")
      }
    })
  ] : []

  postrender = var.is_manager ? {
    binary_path = "${path.module}/kueue-patches/kustomize.sh"
  } : null
}