#!/usr/bin/env bash
cd /home/jichi/yiguan/decompiled/jadx-out/resources/res/layout
echo "=== session / message ==="
ls | grep -iE 'session|msg_|conversation|chat_list'
echo ""
echo "=== flash card ==="
ls | grep -iE 'flash'
echo ""
echo "=== chatroom ==="
ls | grep -iE 'chat_room|chatroom'
echo ""
echo "=== wallet / money / settings ==="
ls | grep -iE 'wallet|withdraw|income|trading|settings|account|security'