# gcp-auth-plugin

## Overview

A Go plugin to generate credentials for Google Kubernetes Engine (GKE) clusters. It can be used as a credentials plugin in a multi-cluster controller using the ClusterProfile API.

## Installation

### Using go install

You can install the plugin in your system using `go install`.

```shell
go install github.com/GoogleCloudPlatform/gke-fleet-management/gcp-auth-plugin
```

### Building a container image

You can also containerize the plugin and make it available to your multi-cluster controller via an `initContainer`.

To build the container image and push to Artifacts Registry, run:
```shell
gcloud artifacts repositories create gcp-auth-plugin \ 
    --repository-format=docker \
    --location=${LOCATION} \
    --project=${PROJECT_ID}

gcloud builds submit --project=${PROJECT_ID} \
   --region=${LOCATION} \
   --config=./cloudbuild.yaml
```

To make the plugin available to your multi-cluster controller, add an `initContainer` to add the plugin to a shared `emptyDir` volume before the controller starts.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: multi-cluster-controller
spec:
  template:
    spec:
      containers:
      - name: multi-cluster-controller
        # ...
        volumeMounts:
        - mountPath: /plugins/
          name: clusterprofile-plugins
      volumes:
      - name: clusterprofile-plugins
        emptyDir: {}
      initContainers:
        - name: add-plugins
          image: ${LOCATION}-docker.pkg.dev/${PROJECT}/gcp-auth-plugin/gcp-auth-plugin:v0.0.1
          command: ["cp"]
          args: 
          - "/gcp-auth-plugin"
          - "/plugins/gcp-auth-plugin"
          volumeMounts:
          - name: clusterprofile-plugins
            mountPath: /plugins/
```

