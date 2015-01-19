#!/bin/bash

make
export HAKMES_PORT=9300
export HAKMES_CASK_BASE=http://localhost:9201
# 1kb
#export HAKMES_CHUNK_SIZE=1024
# 1Mb
#export HAKMES_CHUNK_SIZE=1048576
# 4MB
#export HAKMES_CHUNK_SIZE=4194304
# 16MB (fastest tested on my system for large files)
export HAKMES_CHUNK_SIZE=16777216
# 32MB
#export HAKMES_CHUNK_SIZE=33554432
export HAKMES_DB_PATH=hakmes.db
./hakmes
