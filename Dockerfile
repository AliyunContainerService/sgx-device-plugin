FROM registry.cn-hangzhou.aliyuncs.com/alinux/alinux3 as builder

ARG GOVER=1.21.0
ARG GOPROXY=https://mirrors.aliyun.com/goproxy/,direct

RUN yum install -y wget tar make && yum clean all && \
	wget https://mirrors.aliyun.com/golang/go${GOVER}.linux-amd64.tar.gz && tar -C /usr/local -xzf go${GOVER}.linux-amd64.tar.gz

WORKDIR /src
ADD . /src
RUN export PATH=/usr/local/go/bin:$PATH && make

FROM registry.cn-hangzhou.aliyuncs.com/alinux/alinux3 as image
COPY --from=builder /src/_output/sgx-device-plugin /usr/bin/sgx-device-plugin

ENTRYPOINT ["/usr/bin/sgx-device-plugin"]
