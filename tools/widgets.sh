#!/usr/bin/env bash
cd /home/jichi/yiguan/decompiled/jadx-out/resources/res
echo "=== TitleView layout? ==="
ls -la layout/ | grep -iE 'title|verification'
echo "=== strings: title相关默认文案 ==="
grep -oP '(?<=<string name=")[^"]*title[^"]*' values/strings.xml | head
echo "=== TitleView widget source ==="
find ../sources/club/jijigugu/yiguan/ui/widgets -iname '*Title*' -o -iname '*Verification*' | head