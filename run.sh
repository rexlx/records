#!/bin/bash

export RECORDS_CONFIG=$(pwd)/config.json
export ZINC_API_PWD="impossiblephrase"
export KEY_STORE="https://your-endpoint.here"
go build ./source/main/
./main
