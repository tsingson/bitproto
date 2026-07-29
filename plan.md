# Plan: C <-> Go TCP/IP Verification and Next Steps

Date: 2026-07-25
Status: Paused (ready to resume)

## 1) What Has Been Completed

### A. Payload-length API alignment (C generated code path)
- Moved generated C API usage to explicit payload length handling.
- C examples use returned `size_t` from `Encode*` and `Decode*` and assert expected size.

### B. Go-compiler verification flow
- Go-wrapper compiler build + validation flow integrated and repeatedly verified.
- Cross-language interoperability checks already in place for Python/C/Go `.bin` compatibility.

### C. New TCP/IP case (C client <-> Go server)
- Added case directory: `tests/test_encoding/encoding-cases/tcpip/`.
- Added framing-based C client and Go server:
  - C client: `tests/test_encoding/encoding-cases/tcpip/c/main.c`
  - Go server: `tests/test_encoding/encoding-cases/tcpip/go/main.go`
- Added pytest entry: `tests/test_encoding/test_tcpip.py`.

### D. Functional coverage already implemented
- Normal roundtrip, standard mode.
- Normal roundtrip, optimization mode.
- Reject oversized payload-length header.
- Reject truncated payload.
- Concurrent stress case: multi-client + random fragmented send.

### E. CI integration
- CI workflow includes explicit TCP/IP case step:
  - `.github/workflows/ci.yml`
  - step: `Verify TCP/IP C<->Go framing case`
  - command: `pytest tests/test_encoding/test_tcpip.py -v -s -x`

### F. Documentation
- Added guide:
  - `bitproto-c-to-go-tcpip-guide.md`
- Guide covers framing design, critical safety checks, and Zephyr-facing pitfalls.

## 2) Local Validation Snapshot

Validated successfully in this workspace:
- Build + run tcpip case binaries (C/Go).
- Concurrent stress execution:
  - 8 parallel clients
  - each client 32 rounds
  - random fragmentation send mode
  - server completed with `DONE clients=8 rounds=32`
- Python syntax check:
  - `python3 -m py_compile tests/test_encoding/test_tcpip.py`

Note:
- Local environment did not provide a directly usable `pytest` command at one point;
  targeted runtime checks were executed through direct build/run harness commands.

## 3) Known Risks / Open Items

- Need full CI pass confirmation after all recent changes are pushed.
- Need real Zephyr 4.4.1 integration run (actual board + network stack behavior).
- Current stress parameters are moderate; can add heavier nightly profile.

## 4) Next Test Plan (Immediate)

1. CI confirmation
- Trigger CI and ensure the new TCP/IP test step passes in Python 3.11 lane.
- Confirm no regressions in go-compiler verification steps.

2. Heavier stress profile (host-side)
- Run variants such as:
  - clients=16/32
  - rounds=64/128
  - multiple random seeds
- Capture failure rate, timeout rate, and response latency percentiles.

3. Negative protocol matrix extension
- Add bad magic / bad version / bad flags tests.
- Add zero-length and malformed header boundary tests.

4. Runtime hardening checks
- Ensure server closes malformed connections deterministically.
- Verify no goroutine leak under repeated malformed traffic.

## 5) Zephyr 4.4.1 Real-Case Verification Plan (Next Session)

Target:
- Validate bitproto C endpoint on Zephyr 4.4.1 against Go TCP server with real transport behavior.

### A. Zephyr app setup
1. Create a minimal Zephyr 4.4.1 app using generated C codec and runtime C library.
2. Implement transport adapter (socket/UART gateway depending on board/network path):
   - `read_exact`
   - `write_exact`
3. Keep fixed buffers and strict payload length checks.

### B. Wire protocol checks
1. Use the same frame header contract (magic/version/flags/len).
2. Enforce network byte order for length.
3. Enforce max payload guard before read/decode.

### C. End-to-end tests on real target
1. Basic roundtrip (single client, deterministic payload set).
2. Fragmented-send tests (simulate split writes/reads).
3. Multi-connection burst test (if board/network capacity permits).
4. Malformed frame injection (oversized len, truncated payload, bad magic).

### D. Reliability and safety metrics
1. Connection recovery time after malformed input.
2. Decode failure counter and transport error counter.
3. Memory stability (no growth/leak across long run).
4. CPU budget under stress and watchdog behavior.

### E. Exit criteria
- No silent decode corruption.
- All malformed input paths rejected safely.
- Stable behavior in repeated stress rounds.
- Deterministic reconnect/recovery behavior.

## 6) Resume Commands (Quick Start)

From repo root:

1. Build and quick run case
- `cd tests/test_encoding/encoding-cases/tcpip`
- `make -s --no-print-directory clean`
- `make -s --no-print-directory build-c build-go`

2. Run pytest TCP/IP suite
- `pytest tests/test_encoding/test_tcpip.py -v -s -x`

3. Run go-compiler validations
- `make -C compiler verify-go-compiler-c --no-print-directory -s`
- `make -C compiler verify-go-compiler-go --no-print-directory -s`
- `make -C compiler verify-go-compiler-interop --no-print-directory -s`
