#!/bin/sh
set -eu
cd "$(dirname "$0")"
command -v curl >/dev/null 2>&1 || { echo '先执行 apk add curl ca-certificates'; exit 1; }
case "$(uname -m)" in i386|i486|i586|i686|x86) ;; *) echo '此安装包仅供 Linux x86 32位环境（iSH）'; exit 1;; esac
work=$(mktemp -d ./install.XXXXXX)
trap 'rm -rf "$work"' EXIT
trap 'exit 130' INT
trap 'exit 143' TERM HUP
url=https://github.com/xrefft/cfst-ish/releases/download/v0.1.0-ish.1
curl -fL --retry 2 --connect-timeout 15 --max-time 300 "$url/cfst-ish-linux-386.tar.gz" -o "$work/cfst-ish-linux-386.tar.gz"
curl -fL --retry 2 --connect-timeout 15 --max-time 60 "$url/SHA256SUMS" -o "$work/SHA256SUMS"
(cd "$work"; sha256sum -c SHA256SUMS; tar -xzf cfst-ish-linux-386.tar.gz)
chmod +x "$work/cfst"
mv "$work/cfst" ./cfst
echo '安装完成：sh cfst-ish -sample 20 -stop 11 -t 4 -tlr 0 -dd'
