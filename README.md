# gke-fleet-management

This repository contains tools, code samples, and other resources to help you leverage [Fleet](https://cloud.google.com/kubernetes-engine/fleet-management/docs) for managing your clusters.

## Repository Structure

<!-- markdownlint-disable MD033 -->
<table>

  <tr>
    <th style="text-align: center;">Folder</th>
    <th style="text-align: center;">Description</th>
  </tr>
  <tr>
    <td><a href="./fleet-argocd-plugin">fleet-argocd-plugin</a></td>
    <td>
        <em>(Deprecated)</em> Example code to build and deploy an <a href="https://argo-cd.readthedocs.io/en/latest/operator-manual/applicationset/Generators-Plugin/"><code>ArgoCD plugin generator</code></a> for managing multiple clusters. We recommend using the managed <a href="https://docs.cloud.google.com/kubernetes-engine/fleet-management/docs/generate-inventory-for-integrations">Fleet ClusterProfile Syncer</a> with the <a href="https://github.com/argoproj-labs/clusterprofile-integration-for-argocd">Cluster Profile controller for Argo CD</a> as an alternative.
    </td>
  </tr>
  <tr>
    <td><a href="./fleet-charts">fleet-charts</a></td>
    <td>
        Sample <a href="https://helm.sh/">Helm</a> charts for use with <a href="https://cloud.google.com/kubernetes-engine/fleet-management/docs">fleets</a>, including charts for read-only fleet memberships (<a href="./fleet-charts/gke-connect-agent"><code>gke-connect-agent</code></a>, <a href="./fleet-charts/cloud-logging"><code>cloud-logging</code></a>, <a href="./fleet-charts/system-metrics"><code>system-metrics</code></a>), multi-cluster orchestration, and sample workloads.
    </td>
  </tr>
  <tr>
    <td><a href="./gcp-auth-plugin">gcp-auth-plugin</a></td>
    <td>
        Plugin to generate cluster credentials. It can be used as a credentials plugin in the ClusterProfile API.
    </td>
  </tr>
  <tr>
    <td><a href="./multi-cluster-orchestrator">multi-cluster-orchestrator</a></td>
    <td>
        <em>(Deprecated)</em> Simplifies multi-cluster deployments, optimizes resource utilization, and enhances workload reliability and performance. We recommend <a href="https://kueue.sigs.k8s.io/docs/tasks/manage/setup_multikueue/#setup-multikueue-with-clusterprofile-api">MultiKueue</a> (see <a href="./multikueue-clusterprofile"><code>multikueue-clusterprofile</code></a>) as an alternative.
    </td>
  </tr>
  <tr>
    <td><a href="./multikueue-clusterprofile">multikueue-clusterprofile</a></td>
    <td>
        Quickstart and Terraform modules for setting up <a href="https://kueue.sigs.k8s.io/docs/tasks/manage/setup_multikueue/#setup-multikueue-with-clusterprofile-api">MultiKueue</a> with the ClusterProfile API and GKE Fleet.
    </td>
  </tr>
</table>
<!-- markdownlint-enable MD033 -->

## Get help

Please use the [Issues page](https://github.com/GoogleCloudPlatform/gke-fleet-management/issues) to provide feedback or submit a bug report.

## Disclaimer

This is not an officially supported Google product. The code in this repository is for demonstrative purposes only.


## References
- [Fleet Management Overview](https://cloud.google.com/kubernetes-engine/fleet-management/docs)
- [Fleet Concepts](https://cloud.google.com/kubernetes-engine/fleet-management/docs/fleet-concepts)
