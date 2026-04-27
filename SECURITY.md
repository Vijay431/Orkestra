# Security Policy

## 🔒 Supported Versions

Only the latest commit on `main` receives security patches. There are no versioned releases yet.

| Version | Supported |
|---------|-----------|
| `main` (latest) | ✅ |
| older commits | ❌ |

---

## 🚨 Reporting a Vulnerability

**Do not open a public GitHub Issue for security bugs.**

Use GitHub's private disclosure process:

1. Go to [Security → Report a Vulnerability](https://github.com/Vijay431/Orkestra/security/advisories/new)
2. Describe the issue, steps to reproduce, and potential impact
3. Expect an acknowledgment within **72 hours** and a patch within **14 days** for critical issues

---

## 🔍 What We Check

[`govulncheck`](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck) runs on every CI push and scans all reachable dependency paths against the [Go Vulnerability Database](https://vuln.go.dev/).

---

## 🚫 Out of Scope

The following are operator responsibilities, not Orkestra vulnerabilities:

- Not setting `MCP_TOKEN` (bearer auth is opt-in by design)
- SQLite file permissions on the host
- Network exposure of the MCP port (bind to `127.0.0.1` in production)
- Self-hosted misconfiguration
