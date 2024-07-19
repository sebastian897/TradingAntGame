#!/usr/bin/env bash

cmd=$1

go build -C $cmd -o ../bin/ants$cmd

