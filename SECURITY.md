# Security policy

## Reporting a vulnerability

If you discover a security vulnerability in filament, please report it responsibly.

**Do not open a public GitHub issue for security vulnerabilities.**

Instead, email the maintainer directly. Include:

- A description of the vulnerability
- Steps to reproduce
- The potential impact
- Any suggested fix (if you have one)

## Scope

filament is a CLI tool that reads text files and XML specs. It does not make network requests, does not execute arbitrary code, and does not modify files except when explicitly asked (via `init`, `resolve`, `sync`, `migrate`).

The main attack surface is:
- Malicious spec XML files (could cause excessive memory use during parsing)
- Malicious source files (could cause excessive regex matching)

filament uses Go's standard library for XML parsing and regex matching, which have their own security hardening.

## Supported versions

Only the latest version of filament receives security updates.
