# syntax=docker/dockerfile:1

FROM alpine:latest
RUN apk add libc6-compat
WORKDIR /xdcc
COPY --chown=1000:1000 ./go-xdcc ./go-xdcc
RUN chown 1000:1000 . -R
RUN chmod 777 . -R
CMD ["/xdcc/go-xdcc"]