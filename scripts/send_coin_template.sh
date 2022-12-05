#!/bin/bash

set -x
omnicore-cli sendtoaddress $1  1

omnicore-cli "omni_send" mtowceAw2yeftR1pPg15QcsDqsnSik7Spz "$1"  $2 100

mine.sh

echo asset balance:
omnicore-cli omni_getbalance  $1 $2