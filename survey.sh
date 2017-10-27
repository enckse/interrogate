#!/bin/bash
ENV_FILE="./env"
cwd=$PWD
cd /opt/epiphyte/survey/
if [ -e $ENV_FILE ]; then
    source $ENV_FILE
fi
args="$@"
if [ -z "$args" ]; then
    args=$SURVEY_SETTINGS
fi
exit_code=10
while [ $exit_code -eq 10 ]; do
    echo "loading survey $@"
    python survey.py $@
    exit_code=$?
done
cd $cwd
