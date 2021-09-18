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
ESC=$(printf '%s\n' "$1" | sed -e 's/[]\/$*.^[]/\\&/g');
if [ "$(uname)" == "Darwin" ]; then
    sed -i "" "s/goc2_server/$ESC/" main.go     
elif [ "$(expr substr $(uname -s) 1 5)" == "Linux" ]; then
    sed -i "s/goc2_server/$ESC/" main.go
fi

# Now we build
go build -o /tmp/bin/payload

# Change the string back to original.
if [ "$(uname)" == "Darwin" ]; then
    sed -i "" "s/$ESC/goc2_server/" main.go    
elif [ "$(expr substr $(uname -s) 1 5)" == "Linux" ]; then
    sed -i "s/$ESC/goc2_server/" main.go
fi

# Done
echo "payload located at /tmp/bin/payload c2:$1"