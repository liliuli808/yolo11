#!/usr/bin/env bash
cd /home/jichi/yiguan/decompiled/jadx-out/resources/res/values
echo "=== text sizes ==="
grep -oP '(?<=<dimen name="text_)[^"]+">[^<]+' dimens.xml | head -20
echo "=== size 10/16/20/28/32/64 etc ==="
grep -oP '(?<=<dimen name="size_)[^"]+">[^<]+' dimens.xml | head -30
echo "=== other key colors ==="
for c in pink red color_ffb854 color_ff6f78; do
  v=$(grep -Po "(?<=<color name=\"$c\">)[^<]*" colors.xml)
  echo "$c = $v"
done
echo "=== TabTextSize16 style ==="
grep -A3 'name="TabTextSize16"' styles.xml