# [Experimental] Membership to ClusterProfile syncer

This bash-based syncer regularly pulls GKE Fleet Memberships and writes them as ClusterProfiles into a namespace named after the project id.
It handles creation, update and deletion of the ClusterProfiles. It sets up the GKE Connect Gateway endpoint in annotations so that Clusters can be accessed easily and without requiring additional credentials (as long as Workload Identity is setup in the management cluster).

It can be easily modified to show integrations of GKE Fleet with ClusterProfiles and specific applications.


## Prerequisites

1. Configure AR to be used in docker push

```
gcloud auth configure-docker us-west1-docker.pkg.dev
```

## Install

```
./install.sh
```

## Clean up

```
kubectl delete namespace $(gcloud config get-value project)
```
