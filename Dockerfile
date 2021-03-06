ARG REG=docker.io
FROM ${REG}/library/golang:1.18 AS build
WORKDIR /src
ENV GO111MODULE=on
RUN go install github.com/swaggo/swag/cmd/swag@v1.8.1
COPY go.mod go.sum ./
RUN go mod download

COPY . /src
ARG BUILD_VERSION="local docker"
ARG BUILD_GIT_COMMIT="HEAD"
ARG BUILD_REF="0"
ARG BUILD_DATE=""
RUN chmod +x deploy/update-version.sh  \
    && deploy/update-version.sh version.yaml \
    && make swag check \
    && CGO_ENABLED=0 go build -o wharf-provider-github .

ARG REG=docker.io
FROM ${REG}/library/alpine:3.15 AS final
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=build /src/wharf-provider-github /usr/local/bin/wharf-provider-github
ENTRYPOINT ["/usr/local/bin/wharf-provider-github"]

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
