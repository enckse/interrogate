#!/bin/bash
rm -rf bin/
mkdir -p bin/
cp ../examples/*.yaml .

failed=0
_run() {
    echo "running: $1"
    cat settings.conf | sed "s#example#$1#g" > settings.$1.conf
    pkill survey
    ../bin/survey --config settings.$1.conf &
    curl -s http://localhost:8080/survey/testid > bin/survey.$1.html
    curl -s http://localhost:8080/admin?token=123456 > bin/admin.$1.html
    for f in admin survey; do
        diff -u $f.$1.html bin/$f.$1.html
        if [ $? -ne 0 ]; then
            failed=1
        fi
    done
    sleep 1
    pkill survey
}

for f in $(ls *.yaml); do
    _run $(echo $f | sed "s/\.yaml//g")
done
exit $failed
