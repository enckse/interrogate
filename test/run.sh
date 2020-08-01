#!/bin/bash
rm -rf bin/
mkdir -p bin/
rm -f *.yaml
cp ../configs/*.yaml .

../interrogate --config settings.conf &
sleep 3
pkill interrogate

failed=0
_run() {
    echo "running: $1"
    cat settings.conf | sed "s#example#$1#g" > settings.$1.conf
    pkill interrogate
    ../interrogate --config settings.$1.conf &
    sleep 1
    curl -s http://localhost:8080/survey/testid > bin/survey.$1.html
    curl -s http://localhost:8080/admin -u test:123456 > bin/admin.$1.html
    curl -s http://localhost:8080/snapshot/ -X POST -H 'Content-Type: application/x-www-form-urlencoded; charset=UTF-8' -H 'X-Requested-With: XMLHttpRequest' --data 'session=testid&1=&0=ojioj&2=ijoiojoj&3=High&4=&6=on&7=&8=20.00&9=0&10=ijojiojoijojioi'
    for f in admin survey; do
        file=bin/$f.$1.html
        sed -i "s#<td>test\_.*#<td>uid</td>#" $file
        diff -b -u expect/$f.$1.html $file
        if [ $? -ne 0 ]; then
            failed=1
        fi
    done
    sleep 1
    pkill interrogate
}

did=0
for f in $(ls *.yaml); do
    did=1
    _run $(echo $f | sed "s/\.yaml//g")
done
if [ $did -eq 0 ]; then
    echo "no tests ran..."
    failed=1
fi
../interrogate-stitcher --manifest stitch/test.index.manifest --dir stitch/ --config stitch/run.config.test --out $PWD/bin/results
for f in $(ls expect/results*); do
    diff -b -u $f bin/$(basename $f)
    if [ $? -ne 0 ]; then
        failed=1
    fi
done
test -s bin/results.tar.gz
if [ $? -ne 0 ]; then
    echo "invalid tar"
    failed=1
fi
exit $failed
