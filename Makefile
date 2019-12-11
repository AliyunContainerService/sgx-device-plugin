all: binary

.PHONY: binary
binary: clean
	@echo "PHASE: Building sgx-device-plugin ... "
	mkdir _output/
	GOOS=linux go build -o _output/sgx-device-plugin ./cmd/sgx-device-plugin/*.go

.PHONY: clean
clean:
	@echo 'PHASE: Cleaning ...'
	rm -rf _output &>/dev/null

.PHONY:
lint:

