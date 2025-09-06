#!/bin/bash

baseURL="https://lexer747.github.io/blogs/DATE/"

ColorOff='\033[0m'
Red='\033[0;31m'
Green='\033[0;32m'

echo "Please enter a folder name, this is not the same as the title, this will be the URL used"
read foldername

echo -e "You entered ${Green}$foldername${ColorOff}"
echo -e "This will result in ${Green}$baseURL$foldername.html${ColorOff}"
echo -e "Is this correct? (y/N)"

read yes
if ! [[ $yes == "y" || $yes == "Y" || $yes == "yes" || $yes == "YES" ]]; then
    exit 0
fi

ROOT=$(git rev-parse --show-toplevel)
dir="$ROOT/content/pages/blogs/$foldername"

if [ -d "$dir" ]; then
    echo -e "${Red}Folder already found, cannot create.${ColorOff}"
    exit 1
fi

mkdir "$dir"
mkdir "$dir"/images

echo -n "" > "$dir"/images/.gitkeep
echo -n "" > "$dir"/notes.txt
echo -n "UNKNOWN" > "$dir"/published.content
echo -n "" > "$dir"/revisions.content
echo -n "UNKNOWN Title" > "$dir"/title.content


echo "TODO :)

-----

<br>

-----

#### Footnotes

[acci-ping]: https://github.com/Lexer747/acci-ping" > "$dir"/content.md