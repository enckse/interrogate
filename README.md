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

to configure edit a json definition (examples in the `questions` folder) and place them in `/etc/survey/` with a `.config` extension)

### administration

* the server hosts an admin endpoint `/admin` which will display current manifest information and allow for survey restarts
* additionally the results of the ongoing survey may be rendered as html at `/results`

## development

clone and to build
```
make
```

[![Build Status](https://travis-ci.org/epiphyte/survey.svg?branch=master)](https://travis-ci.org/epiphyte/survey)
