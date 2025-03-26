#!/bin/bash

if [ $# -ne 2 ]; then
    echo "Usage: $0 <id> <port>"
    exit 1
fi

ID=$1
PORT=$2

while true; do
    go run main.go --id $ID --port $PORT
    echo "Process crashed. Restasrting..."
    sleep 2
done
