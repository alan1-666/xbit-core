#!/usr/bin/env python3
from pathlib import Path
import re
import sys

ROOT = Path(__file__).resolve().parents[2]

REQUIRED = [
    "AGENTS.md",
    "docs/harness/CONTROL_PLANE.md",
    "docs/architecture/DOMAIN_BOUNDARIES.md",
    "docs/harness/CHANGE_INTENT_PROTOCOL.md",
    "docs/harness/NO_FALLBACK_POLICY.md",
    "docs/harness/DEPRECATION_POLICY.md",
    "docs/harness/COMPLEXITY_BUDGET.md",
    "docs/harness/REVIEW_GATE.md",
    "docs/harness/DRIFT_LEDGER.md",
    "docs/risk/RISK_INVARIANTS.md",
    "tools/discipline/check_governance_links.py",
    "tools/discipline/check_deprecated_refs.py",
    "tools/discipline/check_no_fallback_patterns.py",
    "tools/discipline/verify.py",
]

AGENT_MENTIONS = [
    "docs/harness/CONTROL_PLANE.md",
    "docs/architecture/DOMAIN_BOUNDARIES.md",
    "docs/harness/CHANGE_INTENT_PROTOCOL.md",
    "docs/harness/REVIEW_GATE.md",
    "tools/discipline/verify.py",
]


def main() -> int:
    missing = [path for path in REQUIRED if not (ROOT / path).exists()]
    if missing:
        for path in missing:
            print(f"missing governance file: {path}", file=sys.stderr)
        return 1

    agents = (ROOT / "AGENTS.md").read_text(encoding="utf-8")
    missing_mentions = [path for path in AGENT_MENTIONS if path not in agents]
    if missing_mentions:
        for path in missing_mentions:
            print(f"AGENTS.md does not route to: {path}", file=sys.stderr)
        return 1

    bad_links = []
    for md in [ROOT / path for path in REQUIRED if path.endswith(".md")]:
        text = md.read_text(encoding="utf-8")
        for match in re.finditer(r"`([^`]+\.md)`", text):
            target = match.group(1)
            if target.startswith("/") or target.startswith("http"):
                continue
            if not (ROOT / target).exists() and not (md.parent / target).exists():
                bad_links.append(f"{md.relative_to(ROOT)} -> {target}")
    if bad_links:
        for link in bad_links:
            print(f"broken governance link: {link}", file=sys.stderr)
        return 1

    print("governance links ok")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
