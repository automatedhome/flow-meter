FROM arm32v7/golang:stretch

COPY qemu-arm-static /usr/bin/
WORKDIR /go/src/github.com/automatedhome/flow-meter
COPY . .
RUN make build

FROM arm32v7/busybox:1.30-glibc

COPY --from=0 /go/src/github.com/automatedhome/flow-meter/flow-meter /usr/bin/flow-meter

ENTRYPOINT [ "/usr/bin/flow-meter" ]
