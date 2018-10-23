survey
===

a basic/primitive survey app for LAN-based survey completion.

## setup

use the epiphyte [repository](https://mirror.epiphyte.network/repos) to install as a package

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

### configure

survey question definitions (json) are stored in `/etc/survey/` and must have a `.config` extension, examples are in the `supporting/` folder in the repository

### administration

* the server hosts an admin endpoint `/admin` which will display current manifest information and allow for survey restarts
* additionally the results of the ongoing survey may be rendered as html at `/results`

Accessing these urls will require a token (e.g. `/results?token=123456`) that will be displayed at survey startup

## development

clone and to build
```
make
```
