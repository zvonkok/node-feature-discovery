#!/bin/bash

set -x
set -e

cd /sys/bus/pci/devices

VENDOR_ID=0x10de

for i in `ls /sys/bus/pci/devices`
do
    VENDOR=`cat $i/vendor`
    CLASS=`cat $i/class`

    if [ $(( ${VENDOR} ^ ${VENDOR_ID} )) == 0 ]; then
        # echo "Found NVIDIA PCI card"
        # 0x300=VGA 0x302=3D
        CLASS_DISPLAY=$(( ${CLASS} & 0xff0000 ))
        if [ $(( ${CLASS_DISPLAY} ^ 0x030000 )) == 0 ]; then
            exit 0
        fi

    fi
done

exit 1

