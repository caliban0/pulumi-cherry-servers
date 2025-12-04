PACK             := cherry-servers
PACKDIR          := sdk

PROJECT          := github.com/caliban0/pulumi-cherry-servers

PROVIDER        := pulumi-${PACK}
PROVIDER_PATH   := provider
VERSION_PATH    := ${PROVIDER_PATH}.Version

PULUMI          := pulumi

SCHEMA_FILE     := ${PROVIDER_PATH}/cmd/${PROVIDER}/schema.json

WORKING_DIR     := $(shell pwd)

# Override during CI using `make [TARGET] PROVIDER_VERSION=""` or by setting a PROVIDER_VERSION environment variable
# Local & branch builds will just used this fixed default version unless specified
PROVIDER_VERSION ?= 1.0.0-alpha.0+dev
# Use this normalised version everywhere rather than the raw input to ensure consistency.
VERSION_GENERIC = $(shell pulumictl convert-version --language generic --version "$(PROVIDER_VERSION)")

.PHONY: provider
provider:
	cd provider && go build -o $(WORKING_DIR)/bin/${PROVIDER} -ldflags "-X ${PROJECT}/${VERSION_PATH}=${VERSION_GENERIC}" $(PROJECT)/${PROVIDER_PATH}/cmd/$(PROVIDER)

$(SCHEMA_FILE): provider
	$(PULUMI) package get-schema $(WORKING_DIR)/bin/${PROVIDER} | \
		jq 'del(.version)' > $(SCHEMA_FILE)

# Generates the SDK code.
codegen: $(SCHEMA_FILE) sdk/dotnet sdk/go sdk/nodejs sdk/python sdk/java

.PHONY: sdk/%
sdk/%: $(SCHEMA_FILE)
	rm -rf $@
	$(PULUMI) package gen-sdk --language $* $(SCHEMA_FILE) --version "${VERSION_GENERIC}"

sdk/java: $(SCHEMA_FILE)
	rm -rf $@
	$(PULUMI) package gen-sdk --language java $(SCHEMA_FILE)

sdk/python: $(SCHEMA_FILE)
	rm -rf $@
	$(PULUMI) package gen-sdk --language python $(SCHEMA_FILE) --version "${VERSION_GENERIC}"
	cp README.md ${PACKDIR}/python/

sdk/dotnet: $(SCHEMA_FILE)
	rm -rf $@
	$(PULUMI) package gen-sdk --language dotnet $(SCHEMA_FILE) --version "${VERSION_GENERIC}"

sdk/go: ${SCHEMA_FILE}
	rm -rf $@
	$(PULUMI) package gen-sdk --language go ${SCHEMA_FILE} --version "${VERSION_GENERIC}"
	cp go.mod ${PACKDIR}/go/pulumi-${PACK}/go.mod
	cd ${PACKDIR}/go/pulumi-${PACK} && \
		go mod edit -module=${PROJECT}/${PACKDIR}/go/pulumi-${PACK} && \
		go mod tidy

# Builds the Python distribution package.
python_sdk: sdk/python
	cp README.md ${PACKDIR}/python/
	cd ${PACKDIR}/python/ && \
		rm -rf ./bin/ ../python.bin/ && cp -R . ../python.bin && mv ../python.bin ./bin && \
		python3 -m venv venv && \
		./venv/bin/python -m pip install build && \
		cd ./bin && \
		../venv/bin/python -m build .

# Install provider plugin from the locally built binary.
install-plugin: codegen
	${PULUMI} plugin install resource ${PROVIDER} ${VERSION_GENERIC} --reinstall -f bin/${PROVIDER}
