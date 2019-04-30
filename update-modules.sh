#!/bin/sh
pushd ./src/commands && go get -u && popd;
find ./src/subcommands -maxdepth 1 -mindepth 1 -type d -print0 | \
    xargs -0 sh -c 'for dir; do (pushd $dir && go get -u && popd); done';
