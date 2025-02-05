#!/bin/sh

cmd="./wikilite --web --log"

cd "$(dirname "$0")" && $cmd && exit 0

cd .. && $cmd && exit 0

exit 1

