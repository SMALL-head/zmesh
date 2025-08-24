GCR_REGISTRY_DIR="/etc/containerd/certs.d/gcr.io"
DOCKER_REGISTRY_DIR="/etc/containerd/certs.d/docker.io"
GHCR_REGISTRY_DIR="/etc/containerd/certs.d/ghcr.io"
CLUSTER_NAME="demo"
for node in $(kind get nodes -n "${CLUSTER_NAME}"); do
    # 检查是否已存在registry配置，如果不存在才写入config_path
    # grep -q 使用退出状态码而非输出内容来判断
    # grep找到匹配时返回0，未找到返回1
    if ! docker exec "${node}" grep -q '\[plugins\."io\.containerd\.grpc\.v1\.cri"\.registry\]' /etc/containerd/config.toml 2>/dev/null; then
        cat <<EOF | docker exec -i "${node}" tee -a /etc/containerd/config.toml
[plugins."io.containerd.grpc.v1.cri".registry]
  config_path = "/etc/containerd/certs.d"
EOF
    fi

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

    # 重启containerd服务
    docker exec "${node}" systemctl restart containerd
done