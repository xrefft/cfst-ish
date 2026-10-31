#!/bin/sh
set -eu
cd "$(dirname "$0")"
case "$(uname -m)" in i386|i486|i586|i686|x86) ;; *) echo '此构建仅供 iSH 的 Linux x86 32位环境'; exit 1;; esac
for cmd in go gcc; do
  command -v "$cmd" >/dev/null 2>&1 || { echo '先执行 apk add go gcc musl-dev'; exit 1; }
done
echo '正在从当前源码构建；iSH 中可能较慢，请保持前台运行。'
export GOMAXPROCS=1 CGO_ENABLED=1 GOARCH=386 GOOS=linux GO386=softfloat
go build -p 1 -trimpath -ldflags='-s -w -linkmode external -extldflags -static' -o cfst.new .
chmod +x cfst.new
mv cfst.new cfst
echo '构建完成。试运行：sh cfst-ish -sample 20 -stop 11 -t 4 -tlr 0 -dd'
