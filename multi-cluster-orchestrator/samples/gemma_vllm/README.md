# Gemma 3 vLLM Sample

In this quickstart, you learn how to deploy the Gemma 3 vLLM inference sample
application as a multi-region [Google Kubernetes Engine (GKE)][0] workload using
[Multi-cluster Orchestrator][18] and [Multi-cluster Gateway][19].

The high demand for GPUs across the industry can sometimes lead to shortages in
public cloud regions. Multi-Cluster Orchestrator can help mitigate these stock
out scenarios by automatically scaling workloads to clusters in different
regions with available GPU resources, and scaling them back when appropriate.
This sample also incorporates the new
[`gke-l7-cross-regional-internal-managed-mc`](https://cloud.google.com/kubernetes-engine/docs/how-to/deploying-multi-cluster-gateways#cross-region-ilb)
 GatewayClass to provide internal Layer 7 load balancing across GKE clusters in
 multiple regions, providing private access to the LLM workload.

Infrastructure as Code (IaC) is a practice of managing and provisioning software
infrastructure resources using code. Terraform is a popular open source IaC tool
that supports a wide range of Cloud services, including GKE. As a GKE platform
administrator, you can use Terraform to standardize configuration of your
Kubernetes clusters and streamline your DevOps workflows. To learn more, see
[Terraform support for GKE][1].

## Objectives

- Deploy the pre-requisite infrastructure, including:
  - Hub GKE cluster running Multi-cluster Orchestrator and Argo CD
  - Worker GKE clusters in multiple-regions
- Deploy the Gemma 3 vLLM sample workload using Multi-cluster Orchestrator and
  Argo CD
- Load test the Gemma using Multi-cluster Gateway to trigger
  multi-cluster scaling

## Before you begin

1. In the Google Cloud console, on the project selector page, select or
  [create a Google Cloud project][2].

1. [Make sure that billing is enabled for your Google Cloud project][3].

1. The following service APIs will be enabled:
   1. Compute Engine
   1. Google Kubernetes Engine
   1. GKE Hub
   1. Connect Gateway
   1. Monitoring
   1. Traffic Director
   1. Multi-cluster Ingress
   1. Multi-cluster Service Discovery

1. In addition to the `Owner` role, your account will need the Service Account
  Token Creator role (`roles/iam.serviceAccountTokenCreator`).

1. Ensure your project has sufficient quota for L4 GPUs. For more information,
  see [About GPUs](https://cloud.google.com/kubernetes-engine/docs/concepts/gpus#gpu-quota)
  and [Allocation quotas](https://cloud.google.com/compute/resource-usage#gpu_quota).

1. Create a [Hugging Face](https://huggingface.co/) account, if you don't already have one.

1. [Get access to the Gemma model](https://cloud.google.com/kubernetes-engine/docs/tutorials/serve-gemma-gpu-vllm#model-access)

1. You should be familiar with the basics of Terraform. You can use the following
resources:
   - [Getting Started with Terraform][5] (video)
   - [Terraform commands][6]

## Prepare the environment

In this tutorial, you should use [Cloud Shell][7] to manage resources
hosted on Google Cloud. Cloud Shell is preinstalled with the software you need
for this tutorial, including [Terraform][8], [kubectl][9], and [gcloud CLI][10].

> [!NOTE]
> If you do not use Cloud Shell, you may need to install
> Terraform, kubectl, and gcloud CLI. You must also set your default
> Terraform project with: `export GOOGLE_CLOUD_PROJECT=PROJECT_ID`.

1. Launch a Cloud Shell session from the Google Cloud console, by clicking Cloud
Shell activation icon Activate Cloud Shell in the Google Cloud console. This
launches a session in the bottom pane of the Google Cloud console.

The service credentials associated with this virtual machine are automatic, so
you do not have to set up or download a service account key.

2. Before you run commands, set your default [project][11] in the Google Cloud CLI
  using the following command:

```
gcloud config set project PROJECT_ID
```

3. Clone the GitHub repository:

```
git clone https://github.com/GoogleCloudPlatform/gke-fleet-management.git --single-branch
```

4. Change to the Gemma vLLM sample directory:

```
cd gke-fleet-management/multi-cluster-orchestrator/samples/gemma_vllm
```

## Review the Terraform file

The [Google Cloud Platform Provider][12] is a plugin that lets you
manage and provision Google Cloud resources using Terraform, HashiCorp's
Infrastructure as Code (IaC) tool. The Google Cloud Platform Provider serves as a
bridge between Terraform configurations and the Google Cloud APIs, letting you
define infrastructure resources, such as virtual machines and networks, in a
declarative manner.

1. Review the following Terraform file:

```
cat 1-infrastructure/main.tf
```

This is the Platform Administrator step, and the file describes the following resources:
  - Service accounts
  - IAM permissions
  - GKE hub cluster
  - GKE worker clusters
  - Custom Metrics Stackdriver Adapter
  - Argo CD
  - Multi-cluster Orchestrator
  - MCO generator plugin for Argo CD
  - Argo CD ClusterProfile Syncer

2. Review the following Terraform file:

```
cat 2-workload/main.tf
```

This is the Application Operator step, and the file describes the following resources:

- A [Helm][15] chart for Gemma 3 vLLM [Argo CD][14] ApplicationSet

3. Hugging Face API Token

Edit `2-workload/main.tf` and set `hf_api_token` your Hugging Face API token.

```
  hf_api_token = "REPLACE_WITH_YOUR_HF_API_TOKEN"
```

> [!IMPORTANT]
> Without a valid Hugging Face API token with access to the Gemma model, the
> deployment will enter a crash loop state.

## Deploy Infrastructure

1. In Cloud Shell, run this command to verify that Terraform is available:

```
terraform version
```

The output should be similar to the following:

```
Terraform v1.10.5
on linux_amd64
```

2. Change Directory:

```
cd 1-infrastructure
```

3. Initialize Terraform:

```
terraform init
```

4. Apply the Terraform configuration:

```
terraform apply
```

Review the plan and when prompted, enter `yes`, to confirm actions.

This command may take around 20 minutes to complete.

The output is similar to the following:

```
Apply complete! Resources: 50 added, 0 changed, 0 destroyed.

Outputs:

argocd = <<EOT

In order to access the server UI you have the following options:

1. kubectl port-forward service/argocd-server -n argocd 8080:443

    and then open the browser on http://localhost:8080 and accept the certificate

2. enable ingress in the values file `server.ingress.enabled` and either
      - Add the annotation for ssl passthrough: https://argo-cd.readthedocs.io/en/stable/operator-manual/ingress/#option-1-ssl-passthrough
      - Set the `configs.params."server.insecure"` in the values file and terminate SSL at your ingress: https://argo-cd.readthedocs.io/en/stable/operator-manual/ingress/#option-2-multiple-ingress-objects-and-hosts


After reaching the UI the first time you can login with username: admin and the random password generated during the installation. You can find the password by running:

kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d

(You should delete the initial secret afterwards as suggested by the Getting Started Guide: https://argo-cd.readthedocs.io/en/stable/getting_started/#4-login-using-the-cli)

EOT
```

## Deploy Application

1. Change Directory:

```
cd ../2-workload
```

2. Initialize Terraform:

```
terraform init
```

3. Apply the Terraform configuration:

```
terraform apply
```

When prompted, enter `yes` to confirm actions.

This command may take a minute to complete.

The output is similar to the following:

```
Apply complete! Resources: 1 added, 0 changed, 0 destroyed.
```

## Verify the application is working

Do the following to confirm the Gemma 3 vLLM server is running correctly:

1. Obtain hub cluster credentials:

```
gcloud container clusters get-credentials mco-hub --region us-central1
```

2. Review the Multi-cluster Orchestrator Placement:

```
kubectl describe MultiKubernetesClusterPlacement gemma-vllm-placement-autoscale -n gemma-server
```

The response should be similar to:

```
Name:         gemma-vllm-placement-autoscale
Namespace:    gemma-server
Labels:       app.kubernetes.io/managed-by=Helm
Annotations:  meta.helm.sh/release-name: gemma-vllm-application
              meta.helm.sh/release-namespace: gemma-server
API Version:  orchestra.multicluster.x-k8s.io/v1alpha1
Kind:         MultiKubernetesClusterPlacement
Spec:
  Rules:
    Type:  all-clusters
    Arguments:
      Regex:  ^fleet-cluster-inventory/mco-cluster-
    Type:     cluster-name-regex
  Scaling:
    Autoscale For Capacity:
      Min Clusters Below Capacity Ceiling:  1
      Use Draining:                         true
      Workload Details:
        Deployment Name:  vllm-gemma-3-1b
        Hpa Name:         gemma-server-autoscale
        Namespace:        gemma-server
Status:
  Clusters:
    Last Transition Time:
    Name:                  mco-cluster-europe-west4
    Namespace:             fleet-cluster-inventory
    State:                 ACTIVE
  Last Addition Time:
Events:                    <none>
```

In this example the application was initially deployed to the `mco-cluster-europe-west4` cluster.

3. Review the application status:

```
kubectl get application -n argocd
```

The response should be similar to:

```
NAME                                                  SYNC STATUS   HEALTH STATUS
fleet-cluster-inventory.mco-cluster-europe-west4-gs   Synced        Healthy
```

> [!TIP]
> If the application isn't yet synced and healthy, wait a few minutes and
> retry the step.

4. Retrieve the Gateway address:

```
kubectl get gateway gemma-server-gateway -n gemma-server -o jsonpath="{.status.addresses[0].value}"
```

> [!IMPORTANT]
> If there is no IP address, retry the retrieval step after a few minutes.

5. Create a bastion host on the project’s default network.

6. Connect to the bastion host:

```
gcloud compute ssh BASTION_NAME
```

7. Export the retrieved Gateway address:

```
GATEWAY_ENDPOINT={YOUR GATEWAY ADDRESS}
```

8. Use `curl` to chat with the model:

```
curl http://${GATEWAY_ENDPOINT}/v1/chat/completions -X POST -H "Content-Type: application/json" -d '{
    "model": "google/gemma-3-1b-it",
    "messages": [
        {
          "role": "user",
          "content": "Why is the sky blue?"
        }
    ]
}'
```

> [!TIP]
> It could take up to 10 minutes for the service to be ready for use.

## Scale the workload across clusters

Do the following to generate load:

1. Create `loadtest.js` file with the following contents:

```
import http from 'k6/http';

export const options = {
  discardResponseBodies: true,

  scenarios: {
    // Baseline load
    baseline: {
      executor: 'constant-arrival-rate',
      rate: 1,
      timeUnit: '1s',
      duration: '45m',
      preAllocatedVUs: 5,
      maxVUs: 150,
    },

    // Test load
    test: {
      executor: 'ramping-arrival-rate',
      stages: [
        // Baseline only
        { target: 0, duration: '1m' },
        // Ramp load up
        { target: 45, duration: '5m' },
        { target: 90, duration: '22m' },
      ],
      preAllocatedVUs: 1000,
      maxVUs: 10000,
    },
  },
};

export default function () {
  const url = `http://${__ENV.GATEWAY_ENDPOINT}/v1/chat/completions`;
  const payload = JSON.stringify({
    "model": "google/gemma-3-1b-it",
    "messages": [
        {
          "role": "user",
          "content": "Why is the sky blue?"
        }
    ]
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
    },

    timeout: '120s'
  };

  http.post(url, payload, params);
}
```

2. Install `k6` on the bastion host:

```
sudo apt update; sudo apt install snapd; sudo snap install snapd; sudo snap install k6
```

3. Use `k6` to generate load:

```
k6 run -e GATEWAY_ENDPOINT=${GATEWAY_ENDPOINT} loadtest.js
```

4. Go to the [GKE Workloads][16] page in the Google Cloud Console:

![Screenshot of the GKE workloads page](images/workloads.png)

5. Observe over 25 minutes as the `vllm-gemma-3-1b` deployment scales to the
  maxReplica limit on the first cluster, and then is deployed to the 2nd and
  then eventually the 3rd cluster.

6. (Optional) You can also monitor the cluster deployments in Argo CD:

![Screenshot of Argo CD](images/argocd.png)

The Terraform output from the Deploy Infrastructure steps detail how to access Argo CD on your cluster.

7. The surge load will terminate after ~28 minutes. Continue to observe as the
  workload scales back down into a single cluster.

## Clean up

To avoid incurring charges to your Google Cloud account for the resources used
on this quickstart, follow these steps.

  1. Run the following command first in `2-workload` and then after ~10 minutes in
  `1-infrastructure` to delete the Terraform resources:

```
terraform destroy --auto-approve
```

## What's next

* Examine the contents of `2-workload/charts/gemma-vllm-application` to observe
how to structure your own application.

[0]: https://cloud.google.com/kubernetes-engine
[1]: https://cloud.google.com/kubernetes-engine/docs/resources/use-terraform-gke
[2]: https://cloud.google.com/resource-manager/docs/creating-managing-projects
[3]: https://cloud.google.com/billing/docs/how-to/verify-billing-enabled#console
[5]: https://www.youtube.com/watch?v=BUPenAjobjw
[6]: https://cloud.google.com/docs/terraform/basic-commands
[7]: https://cloud.google.com/shell
[8]: https://cloud.google.com/docs/terraform/get-started-with-terraform
[9]: https://kubernetes.io/docs/reference/kubectl/
[10]: https://cloud.google.com/sdk/gcloud
[11]: https://support.google.com/cloud/answer/6158840
[12]: https://registry.terraform.io/providers/hashicorp/google/latest/docs
[14]: https://argo-cd.readthedocs.io/
[15]: https://helm.sh/
[16]: https://console.cloud.google.com/kubernetes/workload/overview
[18]: multi-cluster-orchestrator/README.md
[19]: https://cloud.google.com/kubernetes-engine/docs/how-to/deploying-multi-cluster-gateways