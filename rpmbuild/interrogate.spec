Name: interrogate
Version: 1.0.0
Release: 1%{?dist}
Summary: LAN-based surveying tool

License: GPL-3
URL: https://cgit.voidedtech.com/interrogate

BuildRequires: git
BuildRequires: golang
BuildRequires: go-bindata

%description
Simple LAN-based tool to survey users

%build
make interrogate interrogate-stitcher

%files
/etc/interrogate/example.yaml
%config(noreplace)
/etc/interrogate/settings.conf
/lib/systemd/system/interrogate.service
/usr/bin/interrogate
/usr/bin/interrogate-stitcher
/usr/lib/tmpfiles.d/interrogate.conf

%install
install -d $RPM_BUILD_ROOT/etc/interrogate
install -d $RPM_BUILD_ROOT/lib/systemd/system/
install -d $RPM_BUILD_ROOT/usr/bin/
install -d $RPM_BUILD_ROOT/usr/lib/tmpfiles.d/
install -Dm644 configs/example.yaml $RPM_BUILD_ROOT/etc/interrogate/
install -Dm644 configs/settings.conf $RPM_BUILD_ROOT/etc/interrogate/
install -Dm644 configs/systemd/interrogate.service $RPM_BUILD_ROOT/lib/systemd/system/
install -Dm755 interrogate $RPM_BUILD_ROOT/usr/bin/
install -Dm755 interrogate-stitcher $RPM_BUILD_ROOT/usr/bin/
install -Dm644 configs/systemd/interrogate.conf $RPM_BUILD_ROOT/usr/lib/tmpfiles.d/
