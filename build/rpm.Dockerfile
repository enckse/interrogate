FROM fedora:rawhide

RUN dnf update -y && dnf install -y golang go-bindata wget make fedora-packager

ARG SURVEY_VERSION
ENV SURVEY=survey-${SURVEY_VERSION}
ENV VERSION=${SURVEY_VERSION}

RUN wget https://cgit.voidedtech.com/survey/snapshot/${SURVEY}.tar.gz
RUN tar xf ${SURVEY}.tar.gz
RUN mv ${SURVEY} build/

RUN rpmdev-setuptree
RUN rmdir ~/rpmbuild/BUILD/
COPY survey.spec ~/rpmbuild/SPECS/
RUN mv build/ ~/rpmbuild/BUILD

WORKDIR ~/rpmbuild/SPECS/

RUN rpmbuild -bb survey.spec
RUN cp ~/rpmbuild/RPMS/x86_64/*.rpm /rpm/
