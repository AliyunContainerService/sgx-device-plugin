FROM alibaba-cloud-linux-3-registry.cn-hangzhou.cr.aliyuncs.com/alinux3/alinux3:latest
 
ARG REGION_ID=cn-hangzhou
 
RUN yum install -y curl && \
	repo_url=https://enclave-${REGION_ID}.oss-${REGION_ID}.aliyuncs.com/repo/alinux/enclave-expr.repo && \
	yum install -y yum-utils && \
	yum-config-manager --add-repo ${repo_url} && \
	yum install -y libsgx-urts libsgx-uae-service
 
WORKDIR /src
COPY src/hello_world src/enclave.signed.so /src
ENTRYPOINT ["/src/hello_world"]
