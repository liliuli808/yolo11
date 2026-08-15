#!/usr/bin/env bash
cd /home/jichi/yiguan/decompiled/jadx-out/resources/res
for c in colorPrimary colorPrimaryDark colorAccent light_blue blue blue_active c4e4e4e c262626 c808080 cd3d3d3 cececec color_70cea7 color_70cea7_dark white_70 black_70 grayccc gray666 gray333 graycccccc underline underline_dark c33000000 send_progress hint_dark audio_time_dark white transparent c1a1a1a; do
  v=$(grep -Po "(?<=<color name=\"$c\">)[^<]*" values/colors.xml)
  echo "$c = $v"
done
echo "=== window related dimens ==="
grep -E 'name="(view_pager|tab|title|nav|status|padding_top)' values/dimens.xml | head -40