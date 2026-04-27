---
layout: default
title: Examples
parent: Tools
nav_order: 2
permalink: /tools/examples
---

# 🧪 Workflow Examples
{: .no_toc }

End-to-end annotated walkthroughs.
{: .fs-5 .fw-300 }

---

The example workbook lives with the rest of the skill content so LLM agents can load it directly:

> **[`skill/references/examples.md`](https://github.com/Vijay431/Orkestra/blob/main/skill/references/examples.md){: .btn .btn-primary }**

---

## What's Inside

- 🔵 **Bug fix loop** — backlog → claim → comment → update done
- 🌟 **Feature epic** — parent + parallel children, swarming agents
- 🚦 **Sequential pipeline** — `exec_mode=seq` with ordering enforcement
- ⚔️ **Concurrent claim conflict** — what `ERR{code:conflict}` looks like and how to recover
- 🔍 **Search-driven triage** — using `ticket_search` to find related tickets before linking
- 📊 **Visualizing an epic** — `ticket_diagram` output and how Mermaid renders it

Each example shows the exact TOON request/response for every step.

---

## Quick Pattern Lookup

| You want... | Pattern in the doc |
|-------------|--------------------|
| One agent, one task | Bug fix loop |
| Many agents, parallel work | Feature epic |
| Strict ordering | Sequential pipeline |
| Recover from a stale etag | Concurrent claim conflict |
| Find related work | Search-driven triage |

---

## See Also

- **[Workflows page]({{ site.baseurl }}/workflows)** — the patterns at a higher level
- **[API Guide]({{ site.baseurl }}/tools/api-guide)** — parameter reference
- **[Troubleshooting]({{ site.baseurl }}/tools/troubleshooting)** — when things go wrong
