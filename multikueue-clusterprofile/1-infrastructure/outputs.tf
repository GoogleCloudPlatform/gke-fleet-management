output "hub_cluster" {
  value = {
    name     = google_container_cluster.hub.name
    location = google_container_cluster.hub.location
    endpoint = google_container_cluster.hub.endpoint
    ca_cert  = google_container_cluster.hub.master_auth[0].cluster_ca_certificate
  }
}

output "worker_clusters" {
  value = [
    for c in values(google_container_cluster.clusters) : {
      name     = c.name
      location = c.location
      endpoint = c.endpoint
      ca_cert  = c.master_auth[0].cluster_ca_certificate
    }
  ]
}