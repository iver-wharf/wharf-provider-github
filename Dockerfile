FROM golang:1.13.4 AS build
WORKDIR /src
ENV GO111MODULE=on
RUN go get github.com/go-delve/delve/cmd/dlv
RUN go get -u github.com/swaggo/swag/cmd/swag@v1.6.5
COPY . /src
RUN swag init && CGO_ENABLED=0 go build -gcflags "all=-N -l" -o main

FROM alpine:3.13.4 AS final
RUN apk add --no-cache ca-certificates

FROM debian:buster AS final
WORKDIR /app
COPY --from=build /go/bin/dlv /
COPY --from=build /src/main ./
ENTRYPOINT ["/dlv"]
