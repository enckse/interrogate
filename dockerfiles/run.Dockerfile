FROM golang:1.14-alpine3.11 AS buildenv

ARG INTERROGATE_VERSION
ENV INTERROGATE=interrogate-${INTERROGATE_VERSION}

RUN apk --no-cache add build-base wget go go-bindata
RUN wget https://cgit.voidedtech.com/interrogate/snapshot/${INTERROGATE}.tar.gz
RUN tar xf ${INTERROGATE}.tar.gz
RUN mv ${INTERROGATE} build/
WORKDIR build
RUN make clean interrogate interrogate-stitcher

FROM alpine:3.11

COPY --from=buildenv /go/build/interrogate /usr/local/bin/
COPY --from=buildenv /go/build/interrogate-stitcher /usr/local/bin/
RUN mkdir /etc/interrogate
COPY --from=buildenv /go/build/configs/settings.conf /etc/interrogate/
COPY --from=buildenv /go/build/configs/example.yaml /etc/interrogate/

EXPOSE 8080

ENTRYPOINT /usr/local/bin/interrogate --config /etc/interrogate/settings.conf
