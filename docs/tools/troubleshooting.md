---
layout: default
title: Troubleshooting
parent: Tools
nav_order: 3
permalink: /tools/troubleshooting
---

# 🚑 Troubleshooting
{: .no_toc }

Errors, what they mean, how to recover.
{: .fs-5 .fw-300 }

---

The full recovery playbook lives next to the skill content:

> **[`skill/references/troubleshooting.md`](https://github.com/Vijay431/Orkestra/blob/main/skill/references/troubleshooting.md){: .btn .btn-primary }**

---

## TL;DR — Error Code Cheat Sheet

| Code | Cause | First Move |
|------|-------|-----------|
| `not_found` | Wrong ID or ticket archived | Verify with `ticket_search` |
| `conflict` | Stale etag OR already claimed | `ticket_get` → retry with fresh `ua` |
| `seq_blocked` | Predecessor not done | `ticket_children` → finish lower `ord` first |
| `invalid` | Schema violation | Check params against [API Guide]({{ site.baseurl }}/tools/api-guide) |
| `internal` | Server bug | Retry once; check `/health`; file an issue |

---

## Diagnostic Endpoints

```bash
# Health + DB connectivity
curl http://localhost:8080/health

# Skill document (tool metadata as TOON)
curl http://localhost:8080/skill

# Server logs (Docker)
docker compose logs -f orkestra
```

---

## When to File an Issue

- Reproducible `internal` errors
- Errors not in the list above
- Behavior that contradicts the [API Guide]({{ site.baseurl }}/tools/api-guide)

→ [Open an issue](https://github.com/Vijay431/Orkestra/issues/new){: .btn .btn-outline }

For security issues, see the [Security Policy](https://github.com/Vijay431/Orkestra/blob/main/SECURITY.md) instead — don't post them publicly.
