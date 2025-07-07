#!/bin/bash

export LEXER747_DEV="1"

function build() {
    pushd src &> /dev/null
    go mod tidy
    go mod vendor -o vendor/
    go build -mod vendor -o Lexer747.github.io github.com/Lexer747/Lexer747.github.io
    ./Lexer747.github.io
    popd &> /dev/null
}

export TIMEFORMAT="%3lR"
time build