#!/bin/bash
set -ex
ORIGINAL="github.com\/timonwong\/loggercheck"
NEW="github.com\/jlewi\/roboweb\/tracecheck"
find ./ -name "*.go"  -exec  sed -i ".bak" "s/${ORIGINAL}/${NEW}/g" {} ";"
sed -i ".bak" "s/${ORIGINAL}/${NEW}/g" go.mod
find ./ -name "*.bak" -exec rm {} ";"