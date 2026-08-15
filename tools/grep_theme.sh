#!/usr/bin/env bash
cd /home/jichi/yiguan/decompiled/jadx-out/resources/res
echo "=== theme styles ==="
grep -E 'name="(Theme|AppTheme|GTheme|.*Night.*|.*Day.*)' values/styles.xml | head -50
echo "=== count global attrs ==="
grep -c 'global' values/attrs.xml
echo "=== global attrs ==="
grep -oP 'name="[a-zA-Z0-9_]*[gG]lobal[a-zA-Z0-9_]*"' values/attrs.xml | sort -u | head -80
