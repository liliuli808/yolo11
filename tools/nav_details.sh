#!/usr/bin/env bash
cd /home/jichi/yiguan/decompiled/jadx-out/resources/res/navigation
echo "=== action blocks ==="
grep -oP 'action\n[^<]+android:id="@\+id/to_\K[^"]*' nav_index.xml | head
echo "=== all action lines ==="
grep -oP '<action[^>]*' nav_index.xml | head -20
echo "=== slide animations ==="
grep -oP 'anim/slide_\K[a-z_]+' nav_index.xml | sort | uniq -c
echo "=== popEnter/popExit ==="
grep -oP '(popEnter|popExit|enterAnim|exitAnim)="@anim/\K[^"]*' nav_index.xml | sort | uniq -c
echo "=== launchSingleTop / launch options ==="
grep -oP '(launchSingleTop|launchSingleTask|popUpTo|popUpToInclusive)="\K[^"]*' nav_index.xml | sort | uniq -c
