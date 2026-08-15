#!/usr/bin/env bash
cd /home/jichi/yiguan/decompiled/jadx-out/resources/res
awk '/<style name="AppTheme" parent/{f=1} f{print} f&&/<\/style>/{f=0}' values/styles.xml
echo "===AppTheme.Dark==="
awk '/<style name="AppTheme.Dark" parent/{f=1} f{print} f&&/<\/style>/{f=0}' values/styles.xml
