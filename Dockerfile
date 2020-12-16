# Build the manager binary
FROM gcr.io/gcp-runtimes/go1-builder:1.13 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we do not need to re-download as much
# and so that source changes do not invalidate our downloaded layer
RUN /usr/local/go/bin/go mod download

# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY clusterversion/ clusterversion/
COPY collector/ collector/
COPY controllers/ controllers/
COPY crhchttp/ crhchttp/
COPY dirconfig/ dirconfig/
COPY packaging/ packaging/
COPY sources/ sources/
COPY strset/ strset/

# Copy git to inject the commit during build
COPY .git .git
# Build
RUN GIT_COMMIT=$(git rev-list -1 HEAD) && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on \
    /usr/local/go/bin/go build -ldflags "-X controllers.GitCommit=$GIT_COMMIT" -a -o manager main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot

# For terminal access, use this image:
# FROM gcr.io/distroless/base:debug

WORKDIR /
COPY --from=builder /workspace/manager .
USER nonroot:nonroot

ENTRYPOINT ["/manager"]
