#!/usr/bin/env bash
cd /home/jichi/yiguan/decompiled/jadx-out/resources/res/navigation
echo "=== keep_origin context ==="
grep -B4 'keep_origin' nav_index.xml | grep -oP '(to_\w+|actionId)' | head
echo "=== full action sample (first 30 lines) ==="
grep -oP '<action(?:[^>]*>){0,1}' nav_index.xml | head -3
echo "=== transition types count ==="
grep -oP '(enterAnim|exitAnim|popEnterAnim|popExitAnim)="@\K[^"]*' nav_index.xml | sort | uniq -c
echo "=== arg types per destination ==="
grep -A3 '<argument' nav_index.xml | grep -oP 'android:name="\K[^"]*|android:argType="\K[^"]*|android:defaultValue="\K[^"]*' | paste - - - 2>/dev/null | head -20
