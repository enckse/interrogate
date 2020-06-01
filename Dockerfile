FROM golang:1.14-alpine3.11 AS buildenv

ARG SURVEY_VERSION
ENV SURVEY=survey-${SURVEY_VERSION}

RUN apk --no-cache add build-base wget go go-bindata
RUN wget https://cgit.voidedtech.com/survey/snapshot/${SURVEY}.tar.gz
RUN tar xf ${SURVEY}.tar.gz
RUN mv ${SURVEY} build/
WORKDIR build
RUN make clean survey survey-stitcher

FROM alpine:3.11

COPY --from=buildenv /go/build/survey /usr/local/bin/
COPY --from=buildenv /go/build/survey-stitcher /usr/local/bin/
RUN mkdir /etc/survey
COPY --from=buildenv /go/build/configs/settings.conf /etc/survey/
COPY --from=buildenv /go/build/configs/example.yaml /etc/survey/

EXPOSE 8080

ENTRYPOINT /usr/local/bin/survey --config /etc/survey/settings.conf
