#!/bin/bash
cd /opt/epiphyte/survey/
if [ -e env ]; then
    source env
fi
exit_code=1
while [ $exit_code -ne 0 ]; do
    echo "loading survey $@"
    python survey.py $@
    exit_code=$?
done
