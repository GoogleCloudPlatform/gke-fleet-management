#!/bin/bash

# 1. Move into the directory where this script (and kustomization.yaml) lives
cd "$(dirname "$0")"

# 2. Save the Helm output (stdin) to a file named all.yaml IN THIS FOLDER
cat <&0 > all.yaml

# 3. Run kustomize build on the current directory
kustomize build . && rm all.yaml