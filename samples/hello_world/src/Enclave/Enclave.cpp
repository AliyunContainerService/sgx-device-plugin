#include "Enclave.h"
#include "Enclave_t.h" /* print_string */
#include <string.h>

void ecall_hello_from_enclave(char *buf, size_t len)
{
    const char *hello = "Hello world From SGX Enclave!";

    size_t size = len;
    if(strlen(hello) < len)
    {
        size = strlen(hello) + 1;
    }

    memcpy(buf, hello, size - 1);
    buf[size-1] = '\0';
}

