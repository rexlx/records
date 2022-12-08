#!/bin/bash

go build ./source/main/
export RECORDS_CONFIG=$(pwd)/config.json
export ZINC_API_PWD="impossiblephrase"
./main
