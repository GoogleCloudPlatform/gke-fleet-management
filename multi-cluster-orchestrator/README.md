# Multi-Cluster Orchestrator [Preview]

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

## Design

The Multi-Cluster Orchestrator controller runs in a hub cluster and provides an
API allowing the user to define how their workload should be scheduled and
scaled across clusters.

Based on these parameters the system continuously monitors metrics specified by
the user and dynamically makes decisions about which clusters should be used to
run the workload at a given time.

The system builds on the [Cluster Inventory
API](https://github.com/kubernetes-sigs/cluster-inventory-api?tab=readme-ov-file#cluster-inventory-api)
developed by the Kubernetes [SIG
Multicluster](https://multicluster.sigs.k8s.io/), with the [ClusterProfile
API](https://github.com/kubernetes/enhancements/blob/master/keps/sig-multicluster/4322-cluster-inventory/README.md)
defining the available clusters to schedule workloads on. The ClusterProfiles
may be provisioned manually by the user or automatically using a plugin to sync
them from an external source-of-truth such as a cloud provider API.

While Multi-Cluster Orchestrator determines which clusters should run the
workload at a given time, the actual delivery of workloads to clusters is
decoupled from the core Multi-Cluster Orchestrator controller. This makes it
possible for users to integrate their preferred existing systems for application
delivery. For example, Multi-Cluster Orchestrator can be integrated with [Argo
CD](https://argo-cd.readthedocs.io) via an
[ApplicationSet](https://argo-cd.readthedocs.io/en/stable/operator-manual/applicationset/)
[generator
plugin](https://argo-cd.readthedocs.io/en/stable/operator-manual/applicationset/Generators-Plugin/).

The system can be used in tandem with multi-cluster load balancers such as GCP's
[Multi-Cluster
Gateway](https://cloud.google.com/kubernetes-engine/docs/how-to/deploying-multi-cluster-gateways)
to dynamically route incoming traffic to the various clusters.

