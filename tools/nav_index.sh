#!/usr/bin/env bash
cd /home/jichi/yiguan/decompiled/jadx-out/resources/res/navigation
echo "=== destinations ==="
grep -oP 'android:id="@\+id/\K[^"]*' nav_index.xml
echo "=== fragments/layouts ==="
grep -oP 'android:name="\K[^"]*' nav_index.xml
echo "=== actions ==="
grep -oP '<action android:id="@\+id/\K[^"]*' nav_index.xml
