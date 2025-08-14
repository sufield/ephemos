FROM alpine@sha256:bc41182d7ef5ffc53a40b044e725193bc10142a1243f395ee852a8d9730fc2ad # alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY ephemos .
CMD ["./ephemos"]