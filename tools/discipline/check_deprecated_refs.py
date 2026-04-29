#!/usr/bin/env python3
from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[2]

IGNORE_PARTS = {
    ".git",
    ".agents",
    "docs",
    "migrations",
    "node_modules",
    "dist",
    "vendor",
}


def main() -> int:
    warnings = []
    for path in ROOT.rglob("*"):
        if not path.is_file():
            continue
        rel = path.relative_to(ROOT)
        if any(part in IGNORE_PARTS for part in rel.parts):
            continue
        if path.suffix not in {".go", ".sql", ".yaml", ".yml", ".json"}:
            continue
        try:
            lines = path.read_text(encoding="utf-8").splitlines()
        except UnicodeDecodeError:
            continue
        for number, line in enumerate(lines, 1):
            lower = line.lower()
            if "deprecated" in lower or "legacy" in lower:
                warnings.append(f"{rel}:{number}: {line.strip()}")

    if warnings:
        print("deprecated/legacy references found; review against docs/harness/DEPRECATION_POLICY.md")
        for item in warnings:
            print(item)
    else:
        print("no deprecated/legacy references found")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
