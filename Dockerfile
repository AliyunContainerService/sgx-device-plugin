FROM centos:7

COPY _output/sgx-device-plugin /usr/bin/sgx-device-plugin

ENTRYPOINT ["/usr/bin/sgx-device-plugin"]
