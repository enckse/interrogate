#!/bin/bash
cd /opt/epiphyte/survey/
if [ -e env ]; then
    source env
fi
args="$@"
if [ -z "$args" ]; then
    args=$SURVEY_SETTINGS
fi
exit_code=1
while [ $exit_code -ne 0 ]; do
    echo "loading survey $@"
    python survey.py $@
    exit_code=$?
done
