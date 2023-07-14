FROM golang:1.18 AS builder

WORKDIR /go/src/fluent-bit-out-redis/

COPY .git Makefile go.* *.go /go/src/fluent-bit-out-redis/
RUN make

FROM fluent/fluent-bit:2.1.7-debug

COPY --from=builder /go/src/fluent-bit-out-redis/out_redis.so /fluent-bit/bin/
COPY *.conf /fluent-bit/etc/

CMD ["/fluent-bit/bin/fluent-bit", "-c", "/fluent-bit/etc/fluent-bit.conf", "-e", "/fluent-bit/bin/out_redis.so"]
