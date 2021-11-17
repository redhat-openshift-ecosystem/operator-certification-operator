ARG quay_expiration=never
ARG release_tag=0.0.0

# Build the manager binary
FROM golang:1.16 as builder

ARG release_tag

# Copy the go source
COPY . /workspace

WORKDIR /workspace

# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Build
RUN make build RELEASE_TAG=${release_tag}

FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

LABEL \
    com.redhat.component="operator-certification-operator" \
    version="0.0.0" \
    name="operator-certification-operator" \
    License="Apache-2.0" \
    io.k8s.display-name="operator-certification-operator bundle" \
    io.k8s.description="bundle for the operator-certification-operator" \
    summary="This is the bundle for the operator-certification-operator" \
    maintainer="opdev" \
    vendor="Red Hat" \
    release="${release_tag}" \
    description="A Kubernetes operator to provision resources for the operator certification"
    
COPY LICENSE /licenses

ARG quay_expiration

# Define that tags should expire after 1 week. This should not apply to versioned releases.
LABEL quay.expires-after=${quay_expiration}

WORKDIR /
COPY --from=builder /workspace/bin/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]
