# Dockerfile for building "compat" variant of Wasm Image Specification.
FROM scratch

ARG WASM_BINARY_PATH
COPY ${WASM_BINARY_PATH} /plugin.wasm
ENTRYPOINT [ "plugin.wasm" ]