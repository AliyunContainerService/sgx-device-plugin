FROM alibabatee/centos_sgx:7

COPY src/hello_world src/enclave.signed.so /app/

WORKDIR /app

ENTRYPOINT ["/app/hello_world"]
