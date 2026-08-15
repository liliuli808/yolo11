#!/usr/bin/env bash
set -uo pipefail
ps -o pid,ppid,user,lstart,cmd -p 2489525
echo "--- parent ---"
PPID="$(awk '/^PPid:/{print $2}' /proc/2489525/status)"
echo "parent=$PPID"
ps -o pid,ppid,user,cmd -p "$PPID"
echo "--- cwd ---"
ls -la /proc/2489525/cwd
echo "--- environ (head) ---"
cat /proc/2489525/environ | tr '\0' '\n' | grep -E 'ENV|PORT|APP' | head