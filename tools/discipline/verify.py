#!/usr/bin/env python3
from pathlib import Path
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[2]

CHECKS = [
    ["python3", "tools/discipline/check_governance_links.py"],
    ["python3", "tools/discipline/check_no_fallback_patterns.py"],
    ["python3", "tools/discipline/check_deprecated_refs.py"],
]


def run(cmd: list[str]) -> int:
    print(f"+ {' '.join(cmd)}")
    return subprocess.run(cmd, cwd=ROOT).returncode


def main() -> int:
    status = 0
    with_tests = "--with-tests" in sys.argv
    for cmd in CHECKS:
        status = run(cmd) or status
    if with_tests:
        status = run(["go", "test", "./..."]) or status
    return status


if __name__ == "__main__":
    raise SystemExit(main())
