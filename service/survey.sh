#!/bin/bash
ENV_FILE="/etc/survey/environment"
if [ -e $ENV_FILE ]; then
    source $ENV_FILE
fi
args="$@"
if [ -z "$args" ]; then
    args=$SURVEY_SETTINGS
fi
echo "loading survey $args"
survey $args
