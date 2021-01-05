interrogate
===

a basic/primitive survey app for LAN-based survey completion.

# setup

* by default data is saved to `/var/cache/interrogate/`
* configuration is in `/etc/interrogate/`

## run

to run
```
interrogate <args>
```

as a service
```
systemctl enable interrogate.service
```

### configure

survey question definitions (yaml) are stored in `/etc/interrogate/` and must have a `.yaml` extension, examples are in the `configs/` folder in the repository

### administration

* the server hosts an admin endpoint `/admin` which will display current manifest information and allow for survey restarts
* additionally the results of the ongoing survey may be rendered as html at `/results`
* accessing `/admin` endpoints require authentication (basic auth) which is either configured and/or shown at startup

To generate the results file manually (using default caching dir)
```
interrogate-stitcher --dir /var/cache/interrogate/<leaf directory> --auto
```

Alternatively navigate to the folder where the results are stored (e.g. `/var/cache/interrogate/<date>`)
```
interrogate-stitcher --dir $PWD --manifest <date/tag>.index.manifest --config run.config.<date/tag>
```

## development

clone and to build
```
make
```
