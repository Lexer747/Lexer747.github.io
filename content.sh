#!/bin/bash


pushd src &> /dev/null

function init() {
    go build -o Lexer747.github.io github.com/Lexer747/Lexer747.github.io
}

function cleanup() {
    rm Lexer747.github.io
    popd &> /dev/null
}

trap cleanup EXIT

init

while inotifywait -e modify,move,create,delete -r ./../content/; do
    ./Lexer747.github.io
done