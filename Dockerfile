FROM debian:buster

ARG SURVEY_VERSION

RUN apt-get update && apt-get install -y wget make && apt-get clean

RUN cd /tmp && wget  https://lab.voidedtech.com/binaries/survey.${SURVEY_VERSION}-1.tar.gz

RUN cd /tmp && tar xf survey.${SURVEY_VERSION}-1.tar.gz && ./deploy

EXPOSE 8080

ENTRYPOINT /usr/bin/survey --config /etc/survey/settings.conf
