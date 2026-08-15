#!/usr/bin/env bash
set -e
export JAVA_HOME=/home/jichi/android-env/jdk-17.0.20+8
export PATH="$JAVA_HOME/bin:$PATH"
cd /home/jichi/yiguan
JADX=/home/jichi/yiguan/tools/jadx/bin/jadx
APK=/home/jichi/yiguan/一罐.apk
OUT=/home/jichi/yiguan/decompiled/jadx-out
mkdir -p "$OUT"
$JADX --version
echo "=== starting jadx decode ==="
$JADX -d "$OUT" --show-bad-code --no-debug-info --threads-count 4 "$APK" 2>&1 | tail -30
echo "=== jadx finished, exit=$? ==="
