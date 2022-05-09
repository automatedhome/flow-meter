FROM golang:1.18 as builder
 
WORKDIR /go/src/github.com/automatedhome/flow-meter
COPY . .
RUN CGO_ENABLED=0 go build -o flow-meter cmd/main.go

FROM busybox:glibc

COPY --from=builder /go/src/github.com/automatedhome/flow-meter/flow-meter /usr/bin/flow-meter

HEALTHCHECK --timeout=5s --start-period=1m \
  CMD wget --quiet --tries=1 --spider http://localhost:7000/health || exit 1

EXPOSE 7000
ENTRYPOINT [ "/usr/bin/flow-meter" ]
