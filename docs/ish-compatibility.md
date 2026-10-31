# iSH compatibility investigation

## Confirmed in this conversation

- `cfst -h` runs on the user's device. This proves startup only.
- GitHub resolution failed using 10.8.8.18; an explicit query to 223.5.5.5 returned
  an IPv4 answer. DNS failure is separate from a running probe freezing.
- The old generated AWK script broke on a line break after `split(`. That parser
  issue is fixed; it does not diagnose Go or ulimit.
- User reports a stall after some successes; no FD trace, signal trace, Go stack
  dump, or exact iSH build has yet established its root cause.

## Upstream evidence and limits

- https://github.com/XIU2/CloudflareSpeedTest/discussions/321 warns of newer iSH
  incompatibility and specifies Linux 386 (not iPhone's native ARM64 binary).
- https://github.com/ish-app/ish/issues/1230 includes reports that GOMAXPROCS=1 and
  an external static musl-linked 386 build help Go runtime freezes. These are
  reports/workarounds, not proof of this user's exact failure.
- https://github.com/ish-app/ish/wiki/Installing-Golang describes Go limitations.
- Upstream task/tcping.go closes each successful TCP connection. No established
  TCP FD leak was found in that path. Raising ulimit is not a demonstrated fix.
- The concurrent worker read len(p.csv) outside its mutex while other workers
  append. This is a real data race, now fixed; no claim it caused the iSH freeze.
- Delay filtering used break although sorting is loss-rate first, then latency;
  it could discard later qualifying entries. Changed to continue.
- Upstream download estimator is preserved. The separate stability validator demands
  completed transfers and minimum byte counts, not merely a positive speed.

## Applied mitigation

`cfst-ish` starts with GOMAXPROCS=1, and `-ish` performs probes sequentially without
animated progress (or its background refresh goroutine). `-sample N` bounds work
before tests; `-stop N` stops after qualifying latency results, not after download
validation. sample-stable.sh stops only after complete stability qualification.

The preview binary is linux/386, GO386=softfloat, CGO_ENABLED=1, externally linked
with static musl. No system DNS, iSH threading setting or ulimit is modified.
An external watchdog terminates stalled per-IP CFST processes in stable mode.

## Validation boundary

CI covers Go race tests, unit tests, vet, compilation, static 386 build and `-h`.
This is Linux validation, not iSH emulation. iOS background suspension, emulator
syscalls and device-specific hangs require real-device validation. Keep iSH in
foreground. It is not honest to call this a confirmed complete iSH runtime fix.

Run `sh diagnose.sh` for offline environment information; `sh diagnose.sh IP`
adds a maximum 20-second test in default/single-processor modes and samples FD
counts. It writes local logs, never uploads them. Review before sharing.
A descriptor count near the soft limit supports resource-exhaustion diagnosis;
a low count during a freeze calls for Go/emulator investigation instead.
