survey
===

a basic/primitive survey app for LAN-based survey completion.

## setup

use the epiphyte [repository](https://github.com/epiphyte/repository) to install as a package

```
pacman -S survey
```

### files

* by default data is saved to `/var/db/survey/`
* configuration is in `/etc/survey/`

### app

The `app/` folder contains a deployable electron-based client.

The URL for the survey can be put in `%APPDATA%/survey/` for windows or `.config/survey/` in linux:
```
vim survey.txt
---
http://192.168.1.2:8080
```

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

## development

to install necessary dependencies

```
make install
```

[![Build Status](https://travis-ci.org/epiphyte/survey.svg?branch=master)](https://travis-ci.org/epiphyte/survey)
