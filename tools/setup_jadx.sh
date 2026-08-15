#!/usr/bin/env bash
set -e
cd /home/jichi/yiguan/tools
JSON=$(curl -sSL https://api.github.com/repos/skylot/jadx/releases/latest)
URL=$(printf '%s' "$JSON" | grep -oP 'https://github.com/skylot/jadx/releases/download/[^\"]*\.zip' | head -1)
echo "URL=$URL"
curl -sSL -o jadx.zip "$URL"
ls -la jadx.zip
unzip -q -o jadx.zip -d jadx
ls -la jadx/bin/
echo SETUP_DONE
