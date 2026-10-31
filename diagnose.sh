#!/bin/sh
# No config changes. Network probing only when an explicit IP is supplied.
set -eu
cd "$(dirname "$0")"
out=$(mktemp -d ./diagnostic.XXXXXX)
{
  date -u
  uname -a
  printf 'fd_soft='; ulimit -Sn
  printf 'fd_hard='; ulimit -Hn
  if [ -r /etc/alpine-release ]; then cat /etc/alpine-release; fi
  if [ -r /proc/ish/version ]; then cat /proc/ish/version; fi
  command -v curl >/dev/null 2>&1 && curl --version
} > "$out/system.txt" 2>&1
if [ -z "${1:-}" ]; then
  echo "已保存环境信息：$out/system.txt"
  echo '指定一个目标 IP 才会测试网络：sh diagnose.sh 104.16.0.1'
  exit 0
fi
case "$1" in *[!0-9.]*|'') echo '请输入IPv4地址'; exit 1;; esac
ip=$1
bin=./cfst
[ -x "$bin" ] || { echo '先运行 sh install.sh'; exit 1; }
pid=''
trap '[ -z "$pid" ] || kill "$pid" 2>/dev/null || true' EXIT
trap 'exit 130' INT TERM HUP
for mode in default single; do
  echo "测试模式：$mode（最长20秒）"
  if [ "$mode" = single ]; then
    GOMAXPROCS=1 "$bin" -ish -ip "$ip" -t 4 -n 1 -dd -debug -o "$out/$mode.csv" > "$out/$mode.log" 2>&1 &
  else
    "$bin" -ip "$ip" -t 4 -n 1 -dd -debug -o "$out/$mode.csv" > "$out/$mode.log" 2>&1 &
  fi
  pid=$!
  count=0
  while kill -0 "$pid" 2>/dev/null && [ "$count" -lt 20 ]; do
    count=$((count+1))
    if [ -d "/proc/$pid/fd" ]; then
      fds=$(ls "/proc/$pid/fd" 2>/dev/null | wc -l)
      printf '%s %s\n' "$count" "$fds" >> "$out/$mode-fds.txt"
    fi
    sleep 1
  done
  if kill -0 "$pid" 2>/dev/null; then
    echo 'watchdog_timeout' >> "$out/$mode.log"
    kill -KILL "$pid" 2>/dev/null || true
  fi
  code=0
  wait "$pid" || code=$?
  printf 'exit=%s\n' "$code" >> "$out/$mode.log"
  pid=''
done
echo "诊断完成：$out"
echo '包含系统版本、fd数量、进程退出码；不包含密码、订阅、环境变量全集。'
