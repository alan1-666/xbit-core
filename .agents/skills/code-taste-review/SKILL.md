---
name: code-taste-review
description: Review xbit-core changes for simplicity, ownership, unnecessary abstraction, and test fit. Use before committing non-trivial Go or schema changes.
---

# Code Taste Review

## Review Lens

- Does the changed service own this behavior?
- Is the abstraction justified by real duplication or complexity?
- Are error paths explicit and observable?
- Is the test surface proportional to risk?
- Did the change mix refactor and behavior unnecessarily?

## Output

- Intent match:
- Simplicity:
- Boundary fit:
- Test fit:
- Taste score:
