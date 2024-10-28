
project_id=${project_id:-"$(gcloud config get-value project)"}
project_number=$(gcloud projects describe "$project_id" --format "value(projectNumber)")
location=${location:-"us-west1"}
cluster_name=${cluster_name:-"management-cluster"}
membership_name="projects/$project_id/locations/$location/memberships/$cluster_name"
syncerrepo="clusterprofile-syncer"
syncer_docker_image="$location-docker.pkg.dev/$project_id/${syncerrepo}/syncer:latest"
date=$(date) # used to get a unique pod so it forces a rollout

# create the repository if it doesn't exist
gcloud artifacts repositories describe $syncerrepo --location $location --project $project_id || result="$?"
if [[ $result -ne 0 ]]; then
  gcloud artifacts repositories create $syncerrepo --location $location --project $project_id --repository-format=docker
fi

docker build -t $syncer_docker_image .
docker push $syncer_docker_image

# create the management cluster if it doesn't exist
gcloud container clusters describe $cluster_name --location $location --project $project_id || result="$?"
if [[ $result -ne 0 ]]; then
  gcloud container clusters create-auto $cluster_name --location $location --project $project_id --enable-fleet
fi

gcloud container fleet memberships get-credentials $membership_name

# management cluster uses GKE WI, we need to give it access to pull memberships.
gcloud projects add-iam-policy-binding $project_id \
  --member="principal://iam.googleapis.com/projects/$project_number/locations/global/workloadIdentityPools/$project_id.svc.id.goog/subject/ns/$project_id/sa/clusterprofile-syncer" \
  --role="roles/gkehub.viewer" --condition="None"


gcloud projects add-iam-policy-binding $project_id \
  --member="principal://iam.googleapis.com/projects/$project_number/locations/global/workloadIdentityPools/$project_id.svc.id.goog/subject/ns/$project_id/sa/clusterprofile-syncer" \
  --role="roles/gkehub.gatewayReader" --condition="None"

kubectl apply -f cluster-profile-crd.yaml
kubectl create namespace $project_id || true

kubectl delete configmap membership-to-clusterprofile -n $project_id || true
kubectl create configmap membership-to-clusterprofile --from-file membership-to-clusterprofile.sh -n $project_id

rm -f syncer_hydrated.yaml
PROJECT=$project_id MEMBERSHIP_NAME=$membership_name PROJECT_NUMBER=$project_number SYNCER_DOCKER_IMAGE=$syncer_docker_image DATE=$date envsubst < syncer.yaml > syncer_hydrated.yaml

kubectl apply -f syncer_hydrated.yaml -n $project_id
