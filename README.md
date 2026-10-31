# CFST for iSH

在 CFST 源码基础上开发的 iSH 兼容版本：保留 TCP/HTTP 延迟测试、下载测速、IPv4/IPv6 输入和 CSV，新增限量随机抽样、够数即停、顺序测试及下载稳定性验证。仓库公开，可免登录拉取。

## iSH 安装

```sh
apk add git curl ca-certificates
git clone https://github.com/xrefft/cfst-ish.git
cd cfst-ish
sh install.sh
```

`install.sh` 下载 [v0.1.0-ish.1 预览版](https://github.com/xrefft/cfst-ish/releases/tag/v0.1.0-ish.1)并验证 SHA256，无需登录或本机编译。静态 musl Linux 386 包已通过 CI 构建及启动检查，仍需 iSH 网络实测。若需从当前源码构建，执行 `apk add go gcc musl-dev`，再执行 `sh build.sh`。

## 候选上限与最终交付数量（新版）

```sh
GOMAXPROCS=1 ./cfst -ish -sample 100 -target 5 -t 4 -tl 1500 -connect-timeout 3s -ip 104.16.0.0/13
```

最多100个不重复候选，逐个检查4/4连接及下载；下载失败继续补选，最终合格5个即停。默认每IP下载至少1MiB、每轮30秒超时、总响应体预算300MiB；候选或预算耗尽时返回实际数量。`-target`不能和`-stop`、`-dd`、`-httping`组合。`-dn`不控制此模式。`-sl`可设置每轮最低MiB/s；`-verify-rounds 3 -verify-bytes 5242880`启用三轮各5MiB验证，最慢/最快均速比默认至少0.5。只读取指定字节量，无需下载整个大文件；不代表完整代理协议可用。

## 先测基础连接

```sh
sh cfst-ish -sample 20 -stop 11 -t 4 -tlr 0 -tl 1500 -connect-timeout 3s -dd
```

从自带 ip.txt 网段中随机抽最多20个不重复IPv4，四次连接全部成功且平均延迟≤1500ms才计为合格，找到11个即停。候选不足也会结束。`-n`仍表示并发；`-ish`/`-stop`模式顺序执行。

```sh
sh cfst-ish -sample 400 -stop 11 -t 4 -tlr 0 -tl 1500 -dd
```

新增 `-sample` 仅支持 IPv4；不用该选项时仍保留上游 IPv6 功能。`-stop` 是延迟阶段的提前停止，不表示最终下载合格数量。它们与原版同名参数的区别见 `sh cfst-ish -h`。

## 连接 + 下载稳定性验证

```sh
sh sample-stable.sh 400 11
```

- 每个 IP 四次 TCP 全成功，平均延迟≤1500ms。
- 连续三轮完整 HTTPS 下载，每轮≥5MiB，均速≥100KiB/s。
- 最慢轮/最快轮≥0.5；每轮最多90秒，间隔2秒。
- 逐IP启动新进程并加外部超时；找到11个完全合格IP即停。
- 响应体总预算300MiB，不含协议开销。旧curl对未知长度响应可能不是硬上限。

```sh
MIN_MIB=10 ROUNDS=4 MAX_TOTAL_MIB=600 sh sample-stable.sh 400 11
```

结果和逐轮日志保存在 `cfst-stable.*`。此验证反映本次到下载域名的表现；轮间均速比不等于逐秒稳定性，不保证未来稳定，也不验证 VLESS/Trojan 等完整代理协议。

## 更新

```sh
git pull --ff-only
sh install.sh
```

## 排查卡住

```sh
sh diagnose.sh
sh diagnose.sh 104.16.0.1
```

第一个命令不测网络；第二个命令记录默认/单核模式、FD数和退出状态。日志只保存在本机。不会修改DNS或提高ulimit。

已发现的代码问题、修复和仍未验证的 iSH 假设见 [兼容性调查](docs/ish-compatibility.md)。不要把“能显示帮助”视为网络功能完全兼容，也不要把云端Linux测试说成iSH实测。

## 开发

```sh
go test -race ./...
go vet ./...
go build .
```

GitHub Actions 执行测试并构建静态musl的linux/386预览包。保留原版 GPL-3.0 许可证及[上游来源](UPSTREAM.md)。
