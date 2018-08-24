survey
===

a basic/primitive survey app for LAN-based survey completion.

## setup

use the epiphyte [repository](https://github.com/epiphyte/repository) to install as a package

```
pacman -S survey
```

### files

* by default data is saved to `/var/cache/survey/`
* configuration is in `/etc/survey/`

## run

to run
```
survey <args>
```

as a service
```
systemctl enable survey.service
```

edit the `/etc/survey/environment` file to set args for running as a service

### configure

survey question definitions (json) are stored in `/etc/survey/` and must have a `.config` extension, examples are in the `questions/` folder in the repository

### administration

* the server hosts an admin endpoint `/admin` which will display current manifest information and allow for survey restarts
* additionally the results of the ongoing survey may be rendered as html at `/results`

Accessing these urls will require a token (e.g. `/results?token=123456`) that will be displayed at survey startup

## development

clone and to build
```
make
```

[![Build Status](https://travis-ci.org/epiphyte/survey.svg?branch=master)](https://travis-ci.org/epiphyte/survey)
