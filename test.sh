#!/bin/bash

# 判断参数是否为空
if [ -z "$1" ]; then
    echo "Usage: ./test.sh [lab1|lab2|lab3]"
    echo "false"
    exit 1
fi

LAB=$1
STATUS=0

echo "======================================"
echo "    Running Validation for $LAB     "
echo "======================================"

if [ "$LAB" = "lab1" ]; then
    echo "> Testing crypt module..."
    go test ./crypt -v
    if [ $? -ne 0 ]; then STATUS=1; fi

elif [ "$LAB" = "lab2" ]; then
    echo "> Testing types module..."
    go test ./types -v
    if [ $? -ne 0 ]; then STATUS=1; fi
    
    echo "> Testing block module..."
    go test ./block -v
    if [ $? -ne 0 ]; then STATUS=1; fi

elif [ "$LAB" = "lab3" ]; then
    echo "> Testing consensus module..."
    go test ./consensus -v
    if [ $? -ne 0 ]; then STATUS=1; fi

else
    echo "Error: Unknown argument '$LAB'."
    echo "Available choices: lab1, lab2, lab3"
    echo "false"
    exit 1
fi

echo "======================================"
echo "         Validation Result            "
echo "======================================"

if [ $STATUS -eq 0 ]; then
    echo "true"
    exit 0
else
    echo "false"
    exit 1
fi
