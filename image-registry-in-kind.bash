GCR_REGISTRY_DIR="/etc/containerd/certs.d/gcr.io"
DOCKER_REGISTRY_DIR="/etc/containerd/certs.d/docker.io"
GHCR_REGISTRY_DIR="/etc/containerd/certs.d/ghcr.io"
CLUSTER_NAME="demo"
for node in $(kind get nodes -n "${CLUSTER_NAME}"); do
    # gcr.io
    docker exec "${node}" mkdir -p "${GCR_REGISTRY_DIR}"
    docker exec "${node}" touch "${GCR_REGISTRY_DIR}/hosts.toml"
    cat <<EOF |  docker exec -i "${node}" cp /dev/stdin "${GCR_REGISTRY_DIR}/hosts.toml"
[host."https://gcr.m.daocloud.io"]
  capabilities = ["pull", "resolve", "push"]
EOF
    # docker.io
    docker exec "${node}" mkdir -p "${DOCKER_REGISTRY_DIR}"
    docker exec "${node}" touch "${DOCKER_REGISTRY_DIR}/hosts.toml"
    cat <<EOF |  docker exec -i "${node}" cp /dev/stdin "${DOCKER_REGISTRY_DIR}/hosts.toml"
[host."https://docker.m.daocloud.io"]
  capabilities = ["pull", "resolve", "push"]
EOF
    # ghcr.io
    docker exec "${node}" mkdir -p "${GHCR_REGISTRY_DIR}"
    docker exec "${node}" touch "${GHCR_REGISTRY_DIR}/hosts.toml"
    cat <<EOF |  docker exec -i "${node}" cp /dev/stdin "${GHCR_REGISTRY_DIR}/hosts.toml"
[host."https://ghcr.m.daocloud.io"]
  capabilities = ["pull", "resolve", "push"]
EOF
done