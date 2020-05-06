TARGET_IMAGE ?= hello_world

.PHONY: clean
clean:
	@echo "PHASE: Cleaning ..."
	cd src/ && make clean

.PHONY: build
build:
	@echo "PHASE: Building hello_world ..."
	cd src/ && make clean && make

.PHONY: image
image: build
	@echo "PHASE: Building image $(TARGET_IMAGE) ..."
	docker build -t $(TARGET_IMAGE) .

.PHONY:
lint:

