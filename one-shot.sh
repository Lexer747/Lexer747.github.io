#!/bin/bash

export LEXER747_DEV="1"

function build() {
    pushd ../SSG &> /dev/null
    go mod tidy
    go mod vendor -o vendor/
    go build -mod vendor -o ../Lexer747.github.io/Lexer747.github.io github.com/Lexer747/SSG
    popd &> /dev/null
    ./Lexer747.github.io
}

rm -r ./build &> /dev/null
export TIMEFORMAT="%3lR"
time build