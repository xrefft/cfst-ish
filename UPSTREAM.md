# Upstream provenance

This repository contains source from XIU2/CloudflareSpeedTest at commit
`50b42c7f95cfb06413740ce6e1967578ebe63380` (retrieved 2026-10-31).

Original repository: https://github.com/XIU2/CloudflareSpeedTest
Source files: main.go, go.mod, go.sum, task/, utils/, ip.txt, ipv6.txt.
Original GPL-3.0 license is preserved in LICENSE. New additions in this repository
are provided under the same license. This is an independently maintained
compatibility adaptation, not a claim of upstream endorsement.

Changes include sequential iSH mode, bounded unique IPv4 sampling, qualifying
latency-result early stop, concurrent result-count race fix, delay-filter fix. Existing flags retain their upstream meaning;
`-n` remains concurrency, `-sample` is sample size, `-stop` is qualifying count.
