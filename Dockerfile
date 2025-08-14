# Using Alpine Linux 3.20 with pinned SHA256 digest for reproducible builds
FROM alpine:3.20@sha256:4bcff63911fcb4448bd4fdacec207030997caf25e9bea4045fa6c8c44de311d1
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY ephemos .
CMD ["./ephemos"]