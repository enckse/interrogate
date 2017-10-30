#!/bin/bash
_proc() {
for u in $(find $1 -type f | cut -d "/" -f 3 | sort | uniq); do
	match=1
	for f in $(find $1 -type f | grep "$u" | grep "save"); do
		echo $f
		match=0
	done
	if [ $match -eq 1 ]; then
		for f in $(find $1 -type f | grep "$u" | sort -r); do
			echo $f
			break
		done
	fi
done
}

if [ -z "$1" ]; then
	echo "path required"
	exit 1
fi

_proc "$1"
