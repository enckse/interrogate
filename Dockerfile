FROM fedora:latest

RUN dnf -y update && dnf -y install wget make  && dnf clean all

RUN cd /tmp && wget  https://lab.voidedtech.com/binaries/survey.2.7.1-1.tar.gz

RUN cd /tmp && tar xf survey.2.7.1-1.tar.gz && ./deploy

EXPOSE 8080

ENTRYPOINT /usr/bin/survey --config /etc/survey/settings.conf
