#!/bin/bash

pushd src &> /dev/null
go build -mod thirdparty/vendor github.com/Lexer747/Lexer747.github.io
popd &> /dev/null