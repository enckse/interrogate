FROM debian:bullseye

ARG SURVEY_VERSION

RUN apt-get update && apt-get install -y wget make go-bindata golang-go && apt-get clean

RUN wget https://cgit.voidedtech.com/survey/snapshot/survey-${SURVEY_VERSION}.tar.gz
RUN tar xf survey-${SURVEY_VERSION}.tar.gz && mv survey-${SURVEY_VERSION} src
RUN cd src && make survey survey-stitcher

RUN cp src/survey /usr/bin/
RUN cp src/survey-stitcher /usr/bin/
RUN mkdir /etc/survey
RUN cp src/configs/settings.conf /etc/survey/
RUN cp src/configs/example.yaml /etc/survey/

EXPOSE 8080

ENTRYPOINT /usr/bin/survey --config /etc/survey/settings.conf
