ARG quay_expiration=never
ARG release_tag=0.0.0

# Build the manager binary
FROM docker.io/library/golang:1.19 as builder

ARG release_tag

WORKDIR /workspace

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# Copy the code
COPY main.go main.go
COPY Makefile Makefile
COPY hack/ hack/
COPY api/ api/
COPY controllers/ controllers/
COPY internal/ internal/

# Copy git repo for sha info
COPY .git .git

# Build
RUN make build VERSION=${release_tag}

FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

ARG release_tag

LABEL \
    com.redhat.component="operator-certification-operator" \
    version="${release_tag}" \
    name="operator-certification-operator" \
    License="Apache-2.0" \
    io.k8s.display-name="operator-certification-operator bundle" \
    io.k8s.description="bundle for the operator-certification-operator" \
    summary="This is the bundle for the operator-certification-operator" \
    maintainer="opdev" \
    vendor="Red Hat" \
    release="${release_tag}" \
    description="A Kubernetes operator to provision resources for the operator certification"
    
COPY licenses /licenses

ARG quay_expiration

# Define that tags should expire after 1 week. This should not apply to versioned releases.
LABEL quay.expires-after=${quay_expiration}

WORKDIR /
COPY --from=builder /workspace/bin/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]
