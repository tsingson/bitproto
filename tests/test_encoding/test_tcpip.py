import functools
import os
import shutil
import subprocess
from pathlib import Path

import pytest


CASE_DIR = Path(__file__).resolve().parent / "encoding-cases" / "tcpip"
_HAVE_TOOLS = all(shutil.which(t) for t in ("make", "bitproto", "go", "gcc"))
requires_tools = pytest.mark.skipif(
    not _HAVE_TOOLS, reason="needs make, bitproto, go and gcc on PATH"
)


@functools.lru_cache(maxsize=None)
def _build_case(optimized: bool) -> tuple[str, str]:
    mode_suffix = "_opt" if optimized else ""
    build_env = dict(os.environ)

    c_cmd = ["make", "-s", "--no-print-directory", "build-c", f"MODE_SUFFIX={mode_suffix}"]
    if optimized:
        c_cmd.append("OPTIMIZATION_MODE_ARGS=-O")
    subprocess.check_call(c_cmd, cwd=CASE_DIR, env=build_env)
    subprocess.check_call(
        ["make", "-s", "--no-print-directory", "build-go", f"MODE_SUFFIX={mode_suffix}"],
        cwd=CASE_DIR,
        env=build_env,
    )

    return (f"./c/client{mode_suffix}", f"./go/server{mode_suffix}")


def _run_server_client(
    optimized: bool,
    mode: str,
    rounds: int = 1,
    expect_server_success: bool = True,
) -> tuple[subprocess.CompletedProcess[str], int, str]:
    client_bin, server_bin = _build_case(optimized)

    server = subprocess.Popen(
        [server_bin, "-rounds", str(rounds), "-clients", "1"],
        cwd=CASE_DIR,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        text=True,
    )
    assert server.stdout is not None

    ready_line = server.stdout.readline().strip()
    assert ready_line.startswith("READY "), ready_line
    address = ready_line.split(" ", 1)[1]

    client = subprocess.run(
        [client_bin, address, mode, str(rounds)],
        cwd=CASE_DIR,
        capture_output=True,
        text=True,
        check=False,
    )

    server_exit = server.wait(timeout=10)
    server_output = ready_line + "\n" + server.stdout.read()

    if expect_server_success:
        assert server_exit == 0, server_output
    else:
        assert server_exit != 0, server_output

    return client, server_exit, server_output


@requires_tools
@pytest.mark.parametrize("optimized", [False, True])
def test_tcpip_roundtrip(optimized: bool) -> None:
    client, _, server_output = _run_server_client(optimized, "roundtrip", rounds=32)
    assert client.returncode == 0, client.stdout + client.stderr
    assert "DONE clients=1 rounds=32" in server_output
    assert "tcpip client roundtrip ok (32 rounds)" in client.stdout


@requires_tools
def test_tcpip_rejects_oversized_length() -> None:
    client, _, server_output = _run_server_client(
        False, "oversized-length", expect_server_success=False
    )
    assert client.returncode == 0, client.stdout + client.stderr
    assert "invalid payload length" in server_output


@requires_tools
def test_tcpip_rejects_truncated_payload() -> None:
    client, _, server_output = _run_server_client(
        False, "truncated-payload", expect_server_success=False
    )
    assert client.returncode == 0, client.stdout + client.stderr
    assert "read payload" in server_output or "unexpected EOF" in server_output


@requires_tools
@pytest.mark.parametrize("optimized", [False, True])
def test_tcpip_concurrent_fragmented_roundtrip(optimized: bool) -> None:
    clients = 8
    rounds = 32
    client_bin, server_bin = _build_case(optimized)

    server = subprocess.Popen(
        [server_bin, "-rounds", str(rounds), "-clients", str(clients)],
        cwd=CASE_DIR,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        text=True,
    )
    assert server.stdout is not None

    ready_line = server.stdout.readline().strip()
    assert ready_line.startswith("READY "), ready_line
    address = ready_line.split(" ", 1)[1]

    procs = []
    for i in range(clients):
        seed = str(1000 + i)
        procs.append(
            subprocess.Popen(
                [client_bin, address, "fragmented-roundtrip", str(rounds), seed],
                cwd=CASE_DIR,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                text=True,
            )
        )

    outputs = []
    for p in procs:
        out, err = p.communicate(timeout=20)
        outputs.append((p.returncode, out, err))

    server_exit = server.wait(timeout=20)
    server_output = ready_line + "\n" + server.stdout.read()

    assert server_exit == 0, server_output
    assert f"DONE clients={clients} rounds={rounds}" in server_output

    for code, out, err in outputs:
        assert code == 0, out + err
        assert "fragmented roundtrip ok" in out
