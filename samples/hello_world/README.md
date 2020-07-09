# hello_world

`hello_world` is a sample application which demonstrates how to develop and run a SGX application inside docker or 
kubernetes(ACK-TEE), printing messages periodically.

## Build image
This step will build application and then pack it into an image.
```bash
cd sgx-device-plugin/samples/hello_world
TARGET_IMAGE=sgx_hello_world make image
```

## Run it in docker

```bash
docker run -d --name=my_sgx_hello_world --device=/dev/isgx -v /var/run/aesmd/aesm.socket:/var/run/aesmd/aesm.socket sgx_hello_world
docker logs -f my_sgx_hello_world
```


## Run it in Kubernetes(ACK-TEE)

```bash
cat <<EOF | kubectl --kubeconfig kubeconfig create -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: helloworld
  namespace: default
spec:
  replicas: 2
  selector:
    matchLabels:
      app: helloworld
  template:
    metadata:
      labels:
        app: helloworld
    spec:
      containers:
      - command:
        - /app/hello_world
        image: {{TARGET_IMAGE}}
        imagePullPolicy: Always
        name: helloworld
        resources:
          limits:
            cpu: 250m
            memory: 512Mi
            alibabacloud.com/sgx_epc_MiB: 2
        volumeMounts:
        - mountPath: /var/run/aesmd/aesm.socket
          name: aesmsocket
      volumes:
      - hostPath:
          path: /var/run/aesmd/aesm.socket
          type: Socket
        name: aesmsocket
EOF
```

## Clean

```bash
cd sgx-device-plugin/samples/hello_world
make clean
```
