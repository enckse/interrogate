#!/bin/bash
cd /opt/epiphyte/survey/
exit_code=1
while [ $exit_code -ne 0 ]; do
    python survey.py $@
    exit_code=$?
done
