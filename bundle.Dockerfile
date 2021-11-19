ARG quay_expiration=never
ARG release_tag=0.0.0

FROM quay.io/operator-framework/operator-sdk:v1.14.0 as builder

ARG release_tag
# Copy the go source
COPY . /workspace

WORKDIR /workspace

RUN microdnf install -y \
      gcc \
      git

# Generate versioned manifests. when manifests exists this will set the version provided.
RUN make bundle RELEASE_TAG=${release_tag}

FROM scratch

ARG quay_expiration

# Core bundle labels.
LABEL operators.operatorframework.io.bundle.mediatype.v1=registry+v1
LABEL operators.operatorframework.io.bundle.manifests.v1=manifests/
LABEL operators.operatorframework.io.bundle.metadata.v1=metadata/
LABEL operators.operatorframework.io.bundle.package.v1=operator-certification-operator
LABEL operators.operatorframework.io.bundle.channels.v1=alpha
LABEL operators.operatorframework.io.metrics.builder=operator-sdk-v1.14.0
LABEL operators.operatorframework.io.metrics.mediatype.v1=metrics+v1
LABEL operators.operatorframework.io.metrics.project_layout=go.kubebuilder.io/v3

# Labels for testing.
LABEL operators.operatorframework.io.test.mediatype.v1=scorecard+v1
LABEL operators.operatorframework.io.test.config.v1=tests/scorecard/

# Define that tags should expire after 1 week. This should not apply to versioned releases.
LABEL quay.expires-after=${quay_expiration}

# Copy files to locations specified by labels.
COPY --from=builder /workspace/bundle/manifests /manifests/
COPY --from=builder /workspace/bundle/metadata /metadata/
COPY --from=builder /workspace/bundle/tests/scorecard /tests/scorecard/
