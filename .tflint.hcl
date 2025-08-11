plugin "google" {
    enabled = true
    version = "0.35.0"
    source  = "github.com/terraform-linters/tflint-ruleset-google"
}

rule "terraform_required_version" {
  enabled = false
}

rule "terraform_required_providers" {
  enabled = false
}
