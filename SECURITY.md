# Security policy

## Supported versions

Fixes go into the latest release only. The module is at v0, so there are no maintained
older branches.

## Reporting a vulnerability

Please do not open a public issue for a security problem. Report it privately through
[GitHub's vulnerability reporting](https://github.com/t1sk3/go-utils/security/advisories/new)
with:

- the package and function involved,
- a minimal program that shows the problem, and
- what an attacker gains from it.

You can expect a first reply within a week. Once a fix is released, the report is
published as an advisory and credited to you unless you prefer otherwise.

## What counts

This library has no dependencies and does no I/O, so the problems that matter are those
that let input a caller does not control take the program down or hold it up:

- a panic or crash the caller cannot recover from,
- a deadlock or a leaked lock,
- memory or CPU use out of proportion to the size of the input, and
- a data race in a type documented as safe for concurrent use.

Panics that are documented, such as passing a nil function or a non-positive size, are
not vulnerabilities.
