#!/usr/bin/env bash
cd /home/jichi/yiguan/decompiled/jadx-out/resources/res/navigation
awk '/^    <action/{action=$0} /^      android:destination/{print action" -> "$0}' nav_index.xml | sed -E 's/<action[^>]*id="@\+id\/(to_[a-z_]+)"/\1: /; s/<action[^>]*>/ /; s/android:destination="@\/([a-z_]+)"/-> \1/' | head -60
echo "=== count actions ==="
grep -c '<action' nav_index.xml
echo "=== total destinations ==="
grep -c '<fragment' nav_index.xml
