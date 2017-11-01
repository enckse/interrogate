#!/bin/bash
ENV_FILE="/etc/survey.env"
if [ -e $ENV_FILE ]; then
    source $ENV_FILE
fi
args="$@"
if [ -z "$args" ]; then
    args=$SURVEY_SETTINGS
fi
exit_code=10
while [ $exit_code -eq 10 ]; do
    echo "loading survey $args"
    survey $args
    exit_code=$?
done
