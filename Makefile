all: binary

.PHONY: binary
binary: clean
	@echo "PHASE: building sgx-device-plugin ... "
	mkdir _output/
	GOOS=linux go build -o _output/sgx-device-plugin ./cmd/main.go

.PHONY: clean
clean:
	@echo 'PHASE: cleaning ...'
	rm -rf _output &>/dev/null

