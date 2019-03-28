survey
===

a basic/primitive survey app for LAN-based survey completion.

# setup

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

survey question definitions (json) are stored in `/etc/survey/` and must have a `.json` extension, examples are in the `supporting/` folder in the repository

### administration

* the server hosts an admin endpoint `/admin` which will display current manifest information and allow for survey restarts
* additionally the results of the ongoing survey may be rendered as html at `/results`

Accessing these urls will require a token (e.g. `/results?token=123456`) that will be displayed at survey startup

To manually produce html, markdown, or csv outputs, navigate to the folder where the results are stored (e.g. `/var/cache/survey/<date>`)
```
survey-stitcher --dir $PWD --manifest <date/tag>.index.manifest --config run.config.<date/tag>
```

^ will produce all output types by default

## development

clone and to build
```
make
```

Update the `supporting/settings.conf` and adjust the paths to match your file system hierarchy

run it
```
./bin/survey --config supporting/settings.conf
```
