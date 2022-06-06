FROM golang:1.18 as stage

WORKDIR /build

COPY services /build/services
COPY main.go /build/main.go
COPY go.sum /build/go.sum
COPY go.mod /build/go.mod

RUN go test ./... && \
    CGO_ENABLED=0 go build

FROM alpine

LABEL maintainer="zoe zhang https://github.com/zoezhangmattr"
LABEL org.opencontainers.image.source https://github.com/zoezhangmattr/keeper

WORKDIR /opt

COPY --from=stage /build/keeper .

ENTRYPOINT [ "/opt/keeper" ]
