# Contributing to Orkestra

Thanks for taking the time. Here's everything you need to go from zero to a merged PR.

---

## 🛠️ Prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.22+ | Build and test |
| Docker | any | E2E tests only |

---

## 🚀 Getting Started

```bash
git clone https://github.com/Vijay431/Orkestra
cd Orkestra

# Run the server locally
PROJECT_ID=dev DB_PATH=/tmp/dev.db go run ./cmd/server
```

---

## 🧪 Running Tests

```bash
make test          # unit + integration
make test-e2e      # end-to-end (requires nothing extra — uses in-process server)
make test-all      # both
make cover         # generates coverage.html
```

---

## 🔧 Code Conventions

- `go vet ./...` must pass before opening a PR — CI enforces this
- `staticcheck ./...` must pass — CI enforces this too
- New Go files need `// SPDX-License-Identifier: MIT` as the first line
- No `//nolint` without a comment explaining why

**Adding a new MCP tool?** Follow the 4-step guide in [README.md — Adding a New Tool](README.md#adding-a-new-tool) rather than reinventing the pattern.

---

## ✅ Pull Request Checklist

Before opening a PR, confirm:

- [ ] `make test-all` passes
- [ ] `go vet ./...` passes
- [ ] `staticcheck ./...` passes
- [ ] New Go files have the `SPDX-License-Identifier: MIT` header
- [ ] If a tool parameter changed — `skill/references/api-guide.md` is updated
- [ ] If behavior changed — `CHANGELOG.md` has an entry under `[Unreleased]`

---

## 🌿 Branching Model

- **All PRs must target the `dev` branch** — do not open PRs against `main`
- Feature branches should be short-lived and named descriptively: `feat/ticket-reopen`, `fix/etag-comparison`, `docs/update-api-guide`
- `main` is the stable release branch; it receives merges from `dev` only at release time

---

## 📝 Commit Format

Use [Conventional Commits](https://www.conventionalcommits.org/):

```text
feat: add ticket_reopen tool
fix: correct etag comparison on concurrent update
docs: update status codes in SKILL.md
chore: bump mcp-go to v0.50.0
refactor: simplify store.Claim transaction logic
test: add coverage for seq_blocked error path
```

Supported prefixes: `feat`, `fix`, `docs`, `chore`, `refactor`, `test`.

---

## 🐛 Reporting Issues

Open a [GitHub Issue](https://github.com/Vijay431/Orkestra/issues).  
Label it `bug`, `enhancement`, or `question` — it helps triage.

For security issues, see [SECURITY.md](SECURITY.md) instead.
