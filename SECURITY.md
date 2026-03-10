# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability in Kiloforge, please report it responsibly.

**Do not open a public GitHub issue for security vulnerabilities.**

Instead, please email: **security@kiloforge.dev**

Include:
- A description of the vulnerability
- Steps to reproduce the issue
- The potential impact
- Any suggested fixes (if available)

## Response Timeline

- **Acknowledgment:** Within 48 hours of report
- **Initial assessment:** Within 5 business days
- **Fix timeline:** Depends on severity (critical: ASAP, high: 1-2 weeks, medium: next release)

## Scope

This policy applies to:
- The Kiloforge CLI tool (`kf`)
- The orchestrator REST API
- The Command Deck dashboard
- Database handling and credential management

## Out of Scope

- Third-party dependencies (report to the upstream project)
- Issues in development/test configurations only
- Social engineering attacks

## Supported Versions

| Version | Supported |
|---------|-----------|
| Latest release | Yes |
| Previous minor | Security fixes only |
| Older | No |

## Security Best Practices

Kiloforge runs locally on your workstation. To keep your installation secure:

1. Keep Kiloforge updated to the latest version
2. Do not expose the orchestrator port (default 4001) to the network
3. Review agent permissions before granting consent
4. Back up your data directory regularly
