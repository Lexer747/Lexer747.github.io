#!/bin/bash

function build() {
    pushd src &> /dev/null
    go mod tidy
    go mod vendor -o thirdparty/vendor/
    go run -mod vendor github.com/Lexer747/Lexer747.github.io
    popd &> /dev/null
}

export TIMEFORMAT="%3lR"
time build