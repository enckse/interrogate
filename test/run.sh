#!/bin/bash
rm -rf bin/
mkdir -p bin/
pkill survey
../bin/survey --config settings.conf &
curl -s http://localhost:8080/survey/testid > bin/survey.html
curl -s http://localhost:8080/admin?token=123456 > bin/admin.html
failed=0
for f in admin survey; do
    diff -u $f.html bin/$f.html
    if [ $? -ne 0 ]; then
        failed=1
    fi
done
sleep 1
pkill survey
exit $failed
