#!/bin/bash

export LEXER747_DEV="1"

function init() {
    pushd ../SSG &> /dev/null
    go mod tidy
    go mod vendor -o vendor/
    go build -mod vendor -o ../Lexer747.github.io/Lexer747.github.io github.com/Lexer747/SSG
    popd &> /dev/null
}

function cleanup() {
    rm Lexer747.github.io
}

trap cleanup EXIT

init

./Lexer747.github.io

while inotifywait -e modify,move,create,delete -r ./content/; do
    ./Lexer747.github.io
done