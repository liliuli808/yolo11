#!/usr/bin/env bash
cd /home/jichi/yiguan/decompiled/jadx-out/resources/res/layout
echo "=== diary-related ==="
ls | grep -iE 'diary' 
echo ""
echo "=== add/edit ==="
ls | grep -iE 'add|edit' | head -40