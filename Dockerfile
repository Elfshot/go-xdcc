# syntax=docker/dockerfile:1

FROM alpine:latest
WORKDIR /xdcc
COPY --chown=1000:1000 ./go-xdcc ./goxdcc
RUN chown 1000:1000 . -R
RUN chmod 777 . -R
CMD ["/xdcc/goxdcc"]