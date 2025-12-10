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