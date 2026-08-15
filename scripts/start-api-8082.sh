#!/usr/bin/env bash
set -uo pipefail

export HOME=/home/jichi

cd /home/jichi/yiguan/services/api
mkdir -p /home/jichi/yiguan/logs

if ss -ltn 2>/dev/null | grep -q ':8082 '; then
  echo "PORT 8082 OCCUPIED"
  exit 0
fi

nohup ../../.build/api > /home/jichi/yiguan/logs/api.log 2>&1 &
echo "started api pid=$!"

sleep 3
echo "==> health checks"
curl -s --noproxy '*' --max-time 5 http://127.0.0.1:8082/healthz; echo
curl -s --noproxy '*' --max-time 5 http://127.0.0.1:8082/readyz; echo
echo "==> log tail"
tail -8 /home/jichi/yiguan/logs/api.log