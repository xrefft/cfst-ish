#!/bin/sh
# Usage: sh sample.sh [sample_count=20] [success_target=11]
set -eu
N=${1:-20}
export LC_ALL=C
MIN_MIB=${MIN_MIB:-5}
ROUNDS=${ROUNDS:-3}
MIN_KIB=${MIN_KIB:-100}
MIN_RATIO=${MIN_RATIO:-0.5}
ROUND_TIMEOUT=${ROUND_TIMEOUT:-90}
MAX_TOTAL_MIB=${MAX_TOTAL_MIB:-300}
GAP=${GAP:-2}
for value in "$MIN_MIB" "$ROUNDS" "$MIN_KIB" "$ROUND_TIMEOUT" "$MAX_TOTAL_MIB"; do
  case "$value" in ''|*[!0-9]*) echo '阈值必须是正整数'; exit 1;; esac
  [ "$value" -gt 0 ] && [ "$value" -le 100000 ] || exit 1
done
[ "$ROUNDS" -ge 2 ] || { echo '稳定性验证至少需要2轮'; exit 1; }
case "$GAP" in ''|*[!0-9]*) echo 'GAP必须为非负整数'; exit 1;; esac
awk -v r="$MIN_RATIO" 'BEGIN {exit !(r ~ /^[0-9]+([.][0-9]+)?$/ && r>0 && r<=1)}' || { echo 'MIN_RATIO范围为(0,1]'; exit 1; }
BYTES=$((MIN_MIB * 1048576))
BUDGET=$((MAX_TOTAL_MIB * 1048576))
FLOOR=$((MIN_KIB * 1024))
URL=${DOWNLOAD_URL:-https://speed.cloudflare.com/__down?bytes=$BYTES}
case "$URL" in https://*) ;; *) echo '下载地址必须为HTTPS'; exit 1;; esac
AUTHORITY=${URL#https://}; AUTHORITY=${AUTHORITY%%/*}
case "$AUTHORITY" in ''|*:*|*@*|*\?*|*\#*) echo '下载地址须使用普通域名、默认443端口'; exit 1;; esac
HOST=$AUTHORITY
command -v curl >/dev/null 2>&1 || { echo '请先运行：apk add curl ca-certificates'; exit 1; }
TARGET=${2:-11}
for value in "$N" "$TARGET"; do
  case "$value" in ''|*[!0-9]*) echo '请输入正整数'; exit 1;; esac
  [ "$value" -gt 0 ] || exit 1
done
[ "$N" -le 100000 ] || { echo '抽样上限为100000'; exit 1; }
cd "$(dirname "$0")"
export GOMAXPROCS=1
if [ -x ./cfst-ish ]; then BIN=./cfst-ish
elif [ -x ./cfst ]; then BIN=./cfst
elif [ -x ./cfst_linux_386/cfst ]; then BIN=./cfst_linux_386/cfst
else echo '请把本脚本放在 cfst 同目录或其上一级'; exit 1
fi
command -v timeout >/dev/null 2>&1 || { echo '缺少 timeout 命令'; exit 1; }
WORK=$(mktemp -d ./cfst-work.XXXXXX)
trap 'rm -rf "$WORK"' EXIT
trap 'exit 130' INT
trap 'exit 143' TERM HUP
echo "正在生成 $N 个随机候选 IP……"
awk -v want="$N" -v seed="$(date +%s)$$" '
BEGIN {
  srand(seed)
  count=split("173.245.48.0/20 103.21.244.0/22 103.22.200.0/22 103.31.4.0/22 141.101.64.0/18 108.162.192.0/18 190.93.240.0/20 188.114.96.0/20 197.234.240.0/22 198.41.128.0/17 162.158.0.0/15 104.16.0.0/13 104.24.0.0/14 172.64.0.0/13 131.0.72.0/22", nets, " ")
  for (i=1; i<=count; i++) {
    split(nets[i], c, "/")
    split(c[1], a, ".")
    size[i]=2^(32-c[2])
    base[i]=((a[1]*256+a[2])*256+a[3])*256+a[4]
    total+=size[i]
    cumulative[i]=total
  }
  while (got<want && tries++<want*100) {
    r=rand()*total
    j=1
    while (r>=cumulative[j]) j++
    x=base[j]+int(rand()*size[j])
    ip=sprintf("%d.%d.%d.%d", int(x/16777216), int(x/65536)%256, int(x/256)%256, x%256)
    if (!seen[ip]++) { print ip; got++ }
  }
  if (got<want) exit 1
}' > "$WORK/queue"
RESULT=$(mktemp -d ./cfst-stable.XXXXXX)
cp "$WORK/queue" "$RESULT/candidates.txt"
OUT="$RESULT/result.csv"
printf 'ip,tcp_latency_ms,rounds,total_bytes,min_KiB_s,max_KiB_s,min_max_ratio\n' > "$OUT"
printf 'ip,round,curl_exit,http_code,bytes,seconds,bytes_per_second,remote_ip\n' > "$RESULT/downloads.csv"
ok=0
tested=0
used=0
stop=0
echo "目标 $TARGET 个；TCP必须4/4成功，平均延迟不超过1500ms"
echo "下载必须连续 $ROUNDS 轮完成；每轮至少 $MIN_MIB MiB，均速至少 $MIN_KIB KiB/s"
echo "最慢轮/最快轮 >= $MIN_RATIO；总响应体预算 $MAX_TOTAL_MIB MiB"
echo "下载域名：$HOST；结果目录：$RESULT"
while IFS= read -r ip; do
  tested=$((tested+1))
  printf '%s\n' "$ip" > "$WORK/one.txt"
  rm -f "$WORK/one.csv"
  echo "[$tested/$N] $ip：TCP测试……"
  if timeout -k 5 30 "$BIN" -f "$WORK/one.txt" -tp 443 -n 1 -t 4 -tl 1500 -tlr 0 -dd -p 0 -o "$WORK/one.csv" > "$RESULT/last-tcp.log" 2>&1; then :
  else echo '  TCP测试失败或超时'; continue
  fi
  [ -s "$WORK/one.csv" ] || { echo '  TCP未达标'; continue; }
  row=$(tail -n +2 "$WORK/one.csv" | head -n 1)
  [ "${row%%,*}" = "$ip" ] || { echo '  TCP未达标'; continue; }
  latency=$(printf '%s\n' "$row" | cut -d, -f5)
  : > "$WORK/speeds"
  round=1
  passed=1
  ip_bytes=0
  while [ "$round" -le "$ROUNDS" ]; do
    remain=$((BUDGET-used))
    if [ "$remain" -lt "$BYTES" ]; then
      echo '响应体预算不足以完成下一轮，停止测试'
      stop=1; passed=0; break
    fi
    echo "  下载 $round/$ROUNDS：至少 $MIN_MIB MiB……"
    # No redirect, retry or proxy environment; preserve TLS hostname and verify certificate.
    # Range caps supported custom endpoints; max-filesize is an additional guard.
    code=0
    curl -q --silent --show-error --fail --ipv4 --http1.1 \
      --noproxy '*' --resolve "$HOST:443:$ip" \
      --connect-timeout 10 --max-time "$ROUND_TIMEOUT" \
      --speed-limit "$FLOOR" --speed-time 15 \
      --range "0-$((BYTES-1))" --max-filesize "$BYTES" \
      --header 'Accept-Encoding: identity' --header 'Cache-Control: no-cache' \
      --output /dev/null \
      --write-out '%{http_code} %{size_download} %{time_total} %{speed_download} %{remote_ip}\n' \
      "$URL" > "$WORK/metrics" 2> "$RESULT/last-download-error.log" || code=$?
    metrics=$(cat "$WORK/metrics")
    # Account for bodies from failed/partial transfers too.
    actual=$(awk '{if ($2 ~ /^[0-9]+([.][0-9]+)?$/) printf "%.0f",$2; else print 0}' "$WORK/metrics")
    actual=${actual:-0}
    used=$((used+actual))
    ip_bytes=$((ip_bytes+actual))
    printf '%s,%s,%s,' "$ip" "$round" "$code" >> "$RESULT/downloads.csv"
    printf '%s\n' "$metrics" | tr ' ' ',' >> "$RESULT/downloads.csv"
    if [ "$code" -ne 0 ] || ! awk -v b="$BYTES" -v s="$FLOOR" -v ip="$ip" 'BEGIN{ok=0} NF==5 {ok=(($1==200 || $1==206) && $2>=b && $3>0 && $4>=s && $5==ip)} END{exit !ok}' "$WORK/metrics"; then
      echo "  淘汰：下载未完整达标（curl=$code，已收 $actual 字节）"
      passed=0; break
    fi
    awk '{print $4}' "$WORK/metrics" >> "$WORK/speeds"
    awk '{printf "  完成：%.2f MiB，%.1f KiB/s，%.1f 秒\n",$2/1048576,$4/1024,$3}' "$WORK/metrics"
    round=$((round+1))
    if [ "$round" -le "$ROUNDS" ]; then sleep "$GAP"; fi
  done
  [ "$stop" -eq 0 ] || break
  [ "$passed" -eq 1 ] || continue
  if summary=$(awk -v required="$ROUNDS" -v ratio="$MIN_RATIO" 'NR==1{lo=$1;hi=$1} {if($1<lo)lo=$1;if($1>hi)hi=$1} END{if(NR!=required || hi<=0 || lo/hi<ratio)exit 1; printf "%.2f,%.2f,%.4f",lo/1024,hi/1024,lo/hi}' "$WORK/speeds"); then
    printf '%s,%s,%s,%s,%s\n' "$ip" "$latency" "$ROUNDS" "$ip_bytes" "$summary" >> "$OUT"
    ok=$((ok+1))
    echo "  下载稳定性达标：$ok/$TARGET"
    [ "$ok" -lt "$TARGET" ] || break
  else
    echo '  淘汰：各轮均速波动过大'
  fi
done < "$WORK/queue"
echo "结束：测试 $tested 个，合格 $ok 个"
awk -v b="$used" 'BEGIN{printf "累计下载响应体：%.2f MiB（不含协议开销）\n",b/1048576}'
echo "结果：$OUT"
echo "逐轮记录：$RESULT/downloads.csv"
echo '这些结果仅代表本次到下载域名的测试表现，不保证未来稳定或代理协议可用。'
