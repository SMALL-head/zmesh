# 一、使用kind搭建kubernetes集群
kind，全称`kubernetes in docker`，可以帮助我们在一台主机上快速启动Kubernetes集群，是做测试的好工具。  
若没有下载kind，请在参考<a href="https://kind.sigs.k8s.io/#installation-and-usage">kind下载地址</a>下载。务必下载高版本的kind >= v0.7.0，最好用最新的。

```bash
# 创建集群
kind create cluster --config kind.yaml

# docker中各个容器镜像源配置
bash image-registry-in-kin.bash

# 由kind创建的集群的context名字为kind-{yaml中指定的名字}
kubectl cluster-info --context kind-demo

# 集群内部安装cilium网络插件
curl -LO https://github.com/cilium/cilium/archive/main.tar.gz
tar xzf main.tar.gz
cd cilium-main/install/kubernetes
helm install cilium ./cilium \
    --namespace kube-system \
    --set image.pullPolicy=IfNotPresent \
    --set ipam.mode=kubernetes
```

```bash
kubectl get no
# 如果都显示status为Ready，说明网络插件安装成功
# 如果cilium一直安装失败，也可以考虑使用kind内置的网络插件。如果要使用Kind内置的网络插件，则使用 

kind delete cluster --name demo
# 删除集群后，重新创建集群前，修改kind.yaml文件，将disableDefaultCNI这个选项去掉即可

# 然后再执行
kind create cluster --config kind.yaml
```
