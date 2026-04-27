---
layout: default
title: API Guide
parent: Tools
nav_order: 1
permalink: /tools/api-guide
---

# 📘 Full Tool API Reference
{: .no_toc }

Every parameter for every tool. Authoritative source.
{: .fs-5 .fw-300 }

---

The canonical reference lives in the repository so that the running server can serve it via `/skill` and LLM agents can read it directly:

> **[`skill/references/api-guide.md`](https://github.com/Vijay431/Orkestra/blob/main/skill/references/api-guide.md){: .btn .btn-primary }**

It's kept in the repo (and shipped inside the Docker image at `/ORKESTRA_SKILL.md`) so that there's exactly **one** source of truth. This page intentionally doesn't duplicate the content — duplication is how docs go stale.

---

## What's in the Reference

- Required vs optional parameters for every tool
- Default values
- Return shape for each tool (single ticket, array, board, error)
- Validation rules (e.g. `exec_order` uniqueness within a parent)
- Concrete TOON output examples

---

## See Also

- **[Examples]({{ site.baseurl }}/tools/examples)** — end-to-end workflow walkthroughs
- **[Troubleshooting]({{ site.baseurl }}/tools/troubleshooting)** — error code recovery
- **[TOON Format]({{ site.baseurl }}/toon)** — how to read tool responses
