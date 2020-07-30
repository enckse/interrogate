Name: survey
Version: 2.7.3
Release: 1%{?dist}
Summary: LAN-based surveying tool

License: GPL-3
URL: https://cgit.voidedtech.com/survey

BuildRequires: git
BuildRequires: golang
BuildRequires: go-bindata

%description
Simple LAN-based tool to survey users

%build
echo $PWD
make survey survey-stitcher

%files
/etc/survey/example.yaml
%config(noreplace)
/etc/survey/settings.conf
/lib/systemd/system/survey.service
/usr/bin/survey
/usr/bin/survey-stitcher
/usr/lib/tmpfiles.d/survey.conf

%install
install -d $RPM_BUILD_ROOT/etc/survey
install -d $RPM_BUILD_ROOT/lib/systemd/system/
install -d $RPM_BUILD_ROOT/usr/bin/
install -d $RPM_BUILD_ROOT/usr/lib/tmpfiles.d/
install -Dm644 configs/example.yaml $RPM_BUILD_ROOT/etc/survey/
install -Dm644 configs/settings.conf $RPM_BUILD_ROOT/etc/survey/
install -Dm644 configs/systemd/survey.service $RPM_BUILD_ROOT/lib/systemd/system/
install -Dm755 survey $RPM_BUILD_ROOT/usr/bin/
install -Dm755 survey-stitcher $RPM_BUILD_ROOT/usr/bin/
install -Dm644 configs/systemd/survey.conf $RPM_BUILD_ROOT/usr/lib/tmpfiles.d/
