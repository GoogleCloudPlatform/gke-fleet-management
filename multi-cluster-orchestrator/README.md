# Multi-Cluster Orchestrator [Deprecated]

## Project Status

Multi-Cluster Orchestrator is deprecated. We recommend [MultiKueue](https://kueue.sigs.k8s.io/docs/tasks/manage/setup_multikueue/#setup-multikueue-with-clusterprofile-api) as an alternative solution for multi-cluster scheduling.

## About

The Multi-Cluster Orchestrator project provides dynamic scheduling and scaling
of Kubernetes workloads across multiple clusters according to user-defined rules
and metrics.

The primary goals of the project are simplifying multi-cluster deployments,
optimizing resource utilization and costs, and enhancing workload reliability,
scalability, and performance.

One example use case is automatically reacting to capacity exhaustion. For
example, AI/ML workloads require GPUs but demand for GPUs is currently very high
across the industry which can result in public cloud regions temporarily running
out of available GPUs at times. Multi-Cluster Orchestrator can help mitigate
this scenario by automatically scaling the workload out to another cluster in
region which still has available resources then scaling it back in later.

See the [overview page](docs/overview.md) for more details about the project and
its design.

## Samples

- [Gemma 3 vLLM (deploy on Google Cloud using Terraform and Argo CD)](./samples/gemma_vllm/README.md)
- [Hello World (deploy on Google Cloud using Terraform and Argo CD)](./samples/hello_world/README.md)
