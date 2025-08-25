#!/bin/bash

./one-shot.sh

mkdir -p ./bin
cp ./Lexer747.github.io ./bin/Lexer747.github.io
pushd ../SSG || exit
git rev-parse HEAD &> ../Lexer747.github.io/sha.txt
popd || exit

git add ./bin/Lexer747.github.io
git commit -m "Publishing $(date)"
date +"%Y-%m-%d" &> ./published.content

echo "Move the published.content into the correct file"
echo "Commit the update and push to finish the publication"