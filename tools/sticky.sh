#!/usr/bin/env bash
cd /home/jichi/yiguan/decompiled/jadx-out/sources/club/jijigugu/yiguan/ui/widgets/stickypage
ls -la
echo "=== StickyPageBehavior first 80 lines ==="
sed -n '1,80p' StickyPageBehavior.java 2>/dev/null || ls