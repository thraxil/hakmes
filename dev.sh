#!/bin/bash

make
export HAKMES_PORT=9300
export HAKMES_CASK_BASE=http://localhost:9201
export HAKMES_CHUNK_SIZE=1024
export HAKMES_DB_PATH=hakmes.db
./hakmes
