FROM arm32v7/golang:stretch

COPY qemu-arm-static /usr/bin/
WORKDIR /go/src/github.com/automatedhome/flow-meter
COPY . .
RUN make build

FROM arm32v7/busybox:1.30-glibc

COPY --from=0 /go/src/github.com/automatedhome/flow-meter/flow-meter /usr/bin/flow-meter
HEALTHCHECK --timeout=5s --start-period=1m \
  CMD wget --quiet --tries=1 --spider http://localhost:7000/health || exit 1

EXPOSE 7000

ENTRYPOINT [ "/usr/bin/flow-meter" ]
