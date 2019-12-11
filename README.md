# sgx-device-plugin

Kubernetes Device Plugin for Intel SGX

[![Go Report Card](https://goreportcard.com/badge/github.com/AliyunContainerService/sgx-device-plugin)](https://goreportcard.com/report/github.com/AliyunContainerService/sgx-device-plugin)

English | [简体中文](./README-zh_CN.md)

## Overview

`sgx-device-plugin` is a Kubernetes Device Plugin powered by Alibaba Cloud and Ant Financial, making it easier to run SGX applications inside a container.

Intel(R) Software Guard Extensions (Intel(R) SGX) is an Intel technology for application developers seeking to protect select code and data from disclosure or modification. See [official introduction](https://software.intel.com/en-us/sgx) for more details.

## Features

* Using SGX features without privileged mode.
* Support retrieving real EPC size.
* Support EPC resource allocation.

## Prerequisites

* [Intel SGX Drivers](https://github.com/intel/linux-sgx-driver)
* [Intel SGX PSW(Platform Software)](https://github.com/intel/linux-sgx) (If you need AESM)
* Kubernetes version >= 1.10
* Go version >= 1.10

## ACK-TEE Introduction

TEE (Trusted Execution Environment), created by hardware isolation and memory encryption technology such as Intel SGX, is a special execution context named enclave which confidential code and data runs inside. It aims to help application owner to protect their data and prevent data steals by other applications, kernel, BIOS, even all hardware beside CPU.

You could create a confidential Kubernetes cluster using [ACK (Alibaba Cloud Container Service for Kubernetes)](https://www.aliyun.com/product/kubernetes), all worker nodes are running on bare-metal sgx-enabled machines(model: `ecs.ebmhfg5.2xlarge`) which have less overhead, better performance and more stable than VM. By default, containerd, Intel SGX Driver, Intel SGX PSW(Platform Software) and SGX-Device-Plugin will be installed on each node.

## Build

Step 1: Download source code and build binary.

```bash
mkdir -p $GOPATH/src/github.com/AliyunContainerService
git clone https://github.com/AliyunContainerService/sgx-device-plugin.git $GOPATH/src/github.com/AliyunContainerService/sgx-device-plugin
cd $GOPATH/src/github.com/AliyunContainerService/sgx-device-plugin/
make
ls -l _output/sgx-device-plugin
```

Step 2： Build Image.

```bash
docker build -t {SGX_DEVICE_PLUGIN_IMAGE} . -f Dockerfile
docker push {SGX_DEVICE_PLUGIN_IMAGE}
```

## Deployment

While you are creating a confidential Kubernetes cluster using ACK(Alibaba Cloud Container Service for Kubernetes), sgx-device-plugin will be installed by default. Also, you may install it on your own private Kubernetes cluster manually.

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
        scheduler.alpha.Kubernetes.io/critical-pod: ""
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

While plugins are running, run command `kubectl get node {NODE_NAME} -o yaml`, then you will find a new resource type: `alibabacloud.com/sgx_epc_MiB`.

```bash
$ kubectl get node {NODE_NAME} -o yaml
...
  allocatable:
    alibabacloud.com/sgx_epc_MiB: "93"
    cpu: "8"
...
$
```

## Run SGX-Enabled Application

Your application MUST BE SGX-enabled, means that your application is built and signed with SGX SDK, such as Intel SGX SDK.

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

If you want a remote attestation, you should mount `/var/run/aesmd/aesm.socket` in your container, maybe like this:

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

* **Can I deploy this SGX device plugin in my own self-hosting Kubernetes?**  
Yes, this plugin is cloud native, you can run it on sgx-enabled nodes in any Kubernetes.

* **Does this plugin actually limit EPC size for sgx-enabled container?**  
No, EPC size limitation specified by `alibabacloud.com/sgx_epc_MiB` is just used for kube-scheduler.  
Currently, SGX driver doesn't support EPC size limitation.

## License

This software is released under the [Apache 2.0](./LICENSE) license.

## Contributing

See [CONTRIBUTING.md](./docs/en/CONTRIBUTING.md) for details.
