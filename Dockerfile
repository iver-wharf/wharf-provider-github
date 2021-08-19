FROM golang:1.16.5 AS build
WORKDIR /src
ENV GO111MODULE=on

RUN go get -u github.com/swaggo/swag/cmd/swag@v1.7.0
COPY go.mod go.sum ./
RUN go mod download

COPY . /src
ARG BUILD_VERSION="local docker"
ARG BUILD_GIT_COMMIT="HEAD"
ARG BUILD_REF="0"
RUN deploy/update-version.sh version.yaml \
    && make swag \
    && CGO_ENABLED=0 go build -o main

FROM alpine:3.14.0 AS final
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=build /src/main ./
ENTRYPOINT ["/app/main"]

ARG BUILD_VERSION
ARG BUILD_GIT_COMMIT
ARG BUILD_REF
ARG BUILD_DATE
# The added labels are based on this: https://github.com/projectatomic/ContainerApplicationGenericLabels
LABEL name="iver-wharf/wharf-provider-github" \
    url="https://github.com/iver-wharf/wharf-provider-github" \
    release=${BUILD_REF} \
    build-date=${BUILD_DATE} \
    vendor="Iver" \
    version=${BUILD_VERSION} \
    vcs-type="git" \
    vcs-url="https://github.com/iver-wharf/wharf-provider-github" \
    vcs-ref=${BUILD_GIT_COMMIT} \
    changelog-url="https://github.com/iver-wharf/wharf-provider-github/blob/${BUILD_VERSION}/CHANGELOG.md" \
    authoritative-source-url="quay.io"
