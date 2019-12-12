# sgx-device-plugin

支持 Intel SGX 的 Kubernetes 设备插件

[![Go Report Card](https://goreportcard.com/badge/github.com/AliyunContainerService/sgx-device-plugin)](https://goreportcard.com/report/github.com/AliyunContainerService/sgx-device-plugin)

[English](./README.md) | 简体中文

## 介绍

sgx-device-plugin 由阿里云容器服务团队和蚂蚁金服安全计算团队针对 Intel SGX 联合开发的 Kubernetes Device Plugin，可以帮助用户更容易的在容器中使用 SGX。

Intel(R) Software Guard Extensions (Intel(R) SGX) 是 Intel 为软件开发者提供的安全技术，用于防止指定的代码和数据的窃取和恶意篡改。详情可参考[官方链接](https://software.intel.com/en-us/sgx) 。

## 功能

* 无需开启容器特权模式即可使用 SGX；
* 支持 EPC 内存大小自动获取；
* 支持容器声明式 EPC 内存分配。

## 依赖

* [Intel SGX Drivers](https://github.com/intel/linux-sgx-driver)
* [Intel SGX PSW(Platform Software)](https://github.com/intel/linux-sgx) (如果你需要 AESM 服务)
* Kubernetes 版本 >= 1.10
* Go 版本 >= 1.10

## ACK-TEE 简介

TEE (Trusted Execution Environment) ，中文名：可信执行环境，是把用户应用程序代码和数据运行在一个通过硬件孤岛和内存加密技术（Hardware Isolation and memory encryption Technology）创建的特殊执行上线文环境 Enclave 中，任何其他应用、OS Kernel、BIOS、甚至 CPU 之外的其他硬件均无法访问，主要用于防止用户的机密数据、隐私数据被恶意修改、窥探和窃取。

在 [阿里云ACK (Alibaba Cloud Container Service for Kubernetes)](https://www.aliyun.com/product/kubernetes) 上可以创建一个基于 Intel&reg; SGX 的机密计算托管 Kubernetes 集群，节点型号是支持 Intel&reg; SGX 的裸金属服务器 `ecs.ebmhfg5.2xlarge`, 相对于 VM，裸金属的 Overhead 开销更小，性能更优，性能抖动更小。每个节点上默认都会自动安装 containerd、SGX Driver、SGX PSW(Platform Software) 以及 SGX-Device-Plugin。

## 编译&打包镜像（可选）

Step 1: 下载源码并编译。

```bash
mkdir -p $GOPATH/src/github.com/AliyunContainerService
git clone https://github.com/AliyunContainerService/sgx-device-plugin.git $GOPATH/src/github.com/AliyunContainerService/sgx-device-plugin
cd $GOPATH/src/github.com/AliyunContainerService/sgx-device-plugin/
make
ls -l _output/sgx-device-plugin
```

Step 2： 镜像打包

```bash
docker build -t {SGX_DEVICE_PLUGIN_IMAGE} . -f Dockerfile
docker push {SGX_DEVICE_PLUGIN_IMAGE}
```

## 部署 sgx-device-plugin

在创建ACK机密计算托管集群时，默认会自动安装 sgx-device-plugin DaemonSet。当然你也可以在你自己支持 SGX 的 Kubernetes 中选择手动安装:

```bash
$ cat <<EOF | kubectl create -f -
apiVersion: extensions/v1beta1
kind: DaemonSet
metadata:
  name: sgx-device-plugin-ds
  namespace: kube-system
spec:
  selector:
    matchLabels:
      k8s-app: sgx-device-plugin
  template:
    metadata:
      annotations:
        scheduler.alpha.kubernetes.io/critical-pod: ""
      labels:
        k8s-app: sgx-device-plugin
    spec:
      containers:
      - image: registry.cn-hangzhou.aliyuncs.com/acs/sgx-device-plugin:v1.0.0-6e13136-aliyun
        imagePullPolicy: IfNotPresent
        name: sgx-device-plugin
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            drop:
            - ALL
        volumeMounts:
        - mountPath: /var/lib/kubelet/device-plugins
          name: device-plugin
        - mountPath: /dev
          name: dev
      tolerations:
      - effect: NoSchedule
        key: alibabacloud.com/sgx_epc_MiB
        operator: Exists
      volumes:
      - hostPath:
          path: /var/lib/kubelet/device-plugins
          type: DirectoryOrCreate
        name: device-plugin
      - hostPath:
          path: /dev
          type: Directory
        name: dev
EOF
$ kubectl -n kube-system -l k8s-app=sgx-device-plugin
NAME                         READY   STATUS        RESTARTS   AGE
sgx-device-plugin-ds-5brgs   1/1     Running       0          5d5h
sgx-device-plugin-ds-b467q   1/1     Running       0          5d5h
sgx-device-plugin-ds-vl7sm   1/1     Running       0          5d5h
$
```

插件成功运行后，会发现 Node 的 `.status.capacity` 或 `.status.allocatable` 多了一种新资源 **`alibabacloud.com/sgx_epc_MiB`**:

```bash
$ kubectl get node {NODE_NAME} -o yaml
...
  allocatable:
    alibabacloud.com/sgx_epc_MiB: "93"
    cpu: "8"
...
$
```

## 运行 SGX 应用容器

首先你的应用须通过 Intel SGX SDK 或 SGX LibOS 进行编译、签名和镜像打包。

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: {POD_NAME}
  namespace: default
spec:
  containers:
  - image: {CONTAINER_IMAGE}
    imagePullPolicy: IfNotPresent
    name: {CONTAINER_NAME}
    resources:
      requests:
        alibabacloud.com/sgx_epc_MiB: 20
      limits:
        alibabacloud.com/sgx_epc_MiB: 20
```

如果你的应用需要 Remote Attestation，那么需要访问 AESM，这时可以把 `/var/run/aesmd/aesm.socket` 挂载到你的容器中:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: {POD_NAME}
  namespace: default
spec:
  containers:
  - image: {CONTAINER_IMAGE}
    imagePullPolicy: IfNotPresent
    name: {CONTAINER_NAME}
    resources:
      requests:
        alibabacloud.com/sgx_epc_MiB: 20
      limits:
        alibabacloud.com/sgx_epc_MiB: 20
    volumeMounts:
    - mountPath: /var/run/aesmd/aesm.socket
      name: aesmsocket
  volumes:
  - hostPath:
      path: /var/run/aesmd/aesmd/aesm.socket
      type: Socket
    name: aesmsocket

```

## FAQ

* **我可以把这个插件部署到自己的私有 Kubernetes 集群中吗 ?**
当然可以，这个插件是云原生、云平台无关的，你可以把它部署在任何 Kubernetes 上，但它只能运行在 SGX 的节点上。

* **这个插件是否可以真的帮助应用限制 EPC 大小 ？**
不可以，`alibabacloud.com/sgx_epc_MiB` 里指定的 EPC 大小限制仅用于 K8s 的调度, 因为 SGX 驱动目前还不支持 EPC 大小限制。

## 许可证

本软件许可证为 [Apache 2.0](./LICENSE)。

## Contributing

详情请参考 [CONTRIBUTING.md](./docs/en/CONTRIBUTING.md)。
