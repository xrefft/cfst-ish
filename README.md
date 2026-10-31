# cfst-ish

适用于 iSH 的低并发 Cloudflare IPv4 抽样与下载稳定性检查。

## 下载

这是私有仓库。可通过 Git 克隆，或在已有 CFST 的目录里执行：

```sh
apk add curl ca-certificates
curl -fL -u xrefft -H 'Accept: application/vnd.github.raw+json' https://api.github.com/repos/xrefft/cfst-ish/contents/sample-stable.sh -o sample-stable.sh
```

curl 提示密码时，输入具有本仓库 Contents 只读权限的 GitHub personal access token，不是账户密码。不要把 token 写入命令、提交到仓库或发送到聊天。

## 运行

脚本放在 cfst 可执行文件同目录，或包含 cfst_linux_386/cfst 的目录。不包含 CFST 二进制，需自行安装 XIU2/CloudflareSpeedTest。

```sh
sh sample-stable.sh 400 11
```

随机生成最多 400 个不重复 IPv4，逐个测试，找到 11 个合格结果就停止。无需 IP 输入文件。内置网段来自 Cloudflare 官方 IPv4 列表（2026-10-31 快照）。

## 默认合格条件

- TCP 测试 4 次全部成功，平均延迟不超过 1500ms。
- 连续完成 3 轮 HTTPS 下载，每轮至少 5 MiB。
- 每轮均速至少 100 KiB/s，最慢轮/最快轮至少 0.5。
- 每轮最多 90 秒，轮间等待 2 秒；任何一轮失败即淘汰。
- 累计响应体预算 300 MiB；协议开销不计入，某些旧 curl 对未知长度响应的限制可能不是硬上限。

下载请求固定连接候选 IP，并保留下载域名的 TLS 校验；不跟随重定向，不重试，不使用 curl 环境代理。系统级 VPN 仍可能影响路径。默认下载服务为 speed.cloudflare.com。

这些检查仅验证本次到下载服务的表现，不能保证未来稳定，也不验证 VLESS/Trojan 等完整代理协议。轮间均速比不是逐秒波动测量。

## 参数

```sh
MIN_MIB=10 ROUNDS=4 MAX_TOTAL_MIB=600 sh sample-stable.sh 400 11
```

可调环境变量：MIN_MIB、ROUNDS（至少2）、MIN_KIB、MIN_RATIO、ROUND_TIMEOUT、MAX_TOTAL_MIB、GAP。DOWNLOAD_URL 可指定支持 HTTPS 默认443端口的下载文件；文件须足够大，最好支持 Range。更换下载服务后结果代表该服务路径。

结果保存在每次创建的 cfst-stable.* 目录：result.csv、candidates.txt、downloads.csv 和最近一次日志。终端中断会保留结果，临时目录会清理。

## 验证

通过 Shell 语法检查及模拟成功、下载不足、请求失败、速度波动、实际远端 IP 不匹配、流量预算不足等测试。未在用户的 iSH 设备上实测。
