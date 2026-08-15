#!/usr/bin/env bash
set -uo pipefail

export PATH="/home/jichi/.local/bin:/home/jichi/go/bin:/home/jichi/.local/go/bin:$PATH"
export HOME=/home/jichi

cd /home/jichi/yiguan

echo "==> build api"
cd services/api
go build -buildvcs=false -o ../../.build/api ./cmd/api
echo "build ok"

echo "==> migrate up"
cd /home/jichi/yiguan
make migrate-up 2>&1 || echo "migrate returned nonzero (may already be applied)"

if ss -ltn 2>/dev/null | grep -q ':8081 '; then
  echo "PORT 8081 STILL OCCUPIED - skipping start"
  ss -ltnp 2>/dev/null | grep ':8081 '
else
  echo "==> start api"
  cd /home/jichi/yiguan/services/api
  mkdir -p /home/jichi/yiguan/logs
  nohup ../../.build/api > /home/jichi/yiguan/logs/api.log 2>&1 &
  echo "started api pid=$!"
  sleep 2
  curl -s http://127.0.0.1:8081/healthz; echo
  curl -s http://127.0.0.1:8081/readyz; echo
  tail -5 /home/jichi/yiguan/logs/api.log
fi