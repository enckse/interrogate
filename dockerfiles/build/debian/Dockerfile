FROM debian:stable

ARG SURVEY_VERSION
ENV SURVEY=survey-${SURVEY_VERSION}

RUN apt-get update && apt-get upgrade
RUN apt-get install -y wget golang go-bindata build-essential make
RUN wget https://cgit.voidedtech.com/survey/snapshot/${SURVEY}.tar.gz
RUN tar xf ${SURVEY}.tar.gz
RUN mv ${SURVEY} build/
WORKDIR build
RUN dpkg-buildpackage -us -uc --build=binary
