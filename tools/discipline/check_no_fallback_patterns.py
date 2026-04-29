#!/usr/bin/env python3
from pathlib import Path
import re
import sys

ROOT = Path(__file__).resolve().parents[2]

IGNORE_PARTS = {
    ".git",
    ".idea",
    ".agents",
    "docs",
    "migrations",
    "node_modules",
    "dist",
    "vendor",
}
IGNORE_SUFFIXES = {".md", ".sum", ".mod"}
TEXT_SUFFIXES = {".go", ".sql", ".yaml", ".yml", ".json", ".toml", ".env", ".example"}

SUSPICIOUS = [
    re.compile(r"\bfallback\b.*\b(just in case|unknown caller|temporary|silent|silently)\b", re.I),
    re.compile(r"\b(best[- ]effort|best effort)\b.*\b(order|sign|wallet|nonce|leverage|withdraw|transfer)\b", re.I),
    re.compile(r"\bcompat(ibility)?\b.*\bunknown caller\b", re.I),
]


def iter_files():
    for path in ROOT.rglob("*"):
        if not path.is_file():
            continue
        rel = path.relative_to(ROOT)
        if any(part in IGNORE_PARTS for part in rel.parts):
            continue
        if path.suffix in IGNORE_SUFFIXES:
            continue
        if path.suffix and path.suffix not in TEXT_SUFFIXES:
            continue
        yield path


def main() -> int:
    blocked = []
    fallback_mentions = 0
    for path in iter_files():
        try:
            lines = path.read_text(encoding="utf-8").splitlines()
        except UnicodeDecodeError:
            continue
        for number, line in enumerate(lines, 1):
            if "governance:allow-fallback" in line:
                continue
            if "fallback" in line.lower():
                fallback_mentions += 1
            for pattern in SUSPICIOUS:
                if pattern.search(line):
                    blocked.append(f"{path.relative_to(ROOT)}:{number}: {line.strip()}")

    if blocked:
        print("unauthorized fallback patterns found:", file=sys.stderr)
        for item in blocked:
            print(item, file=sys.stderr)
        return 1

    print(f"no blocked fallback patterns found ({fallback_mentions} fallback mentions reviewed)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
