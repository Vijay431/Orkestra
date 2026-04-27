---
layout: default
title: Tools
nav_order: 4
has_children: true
permalink: /tools/
---

# 🛠️ MCP Tools
{: .no_toc }

13 tools, three categories. Pick the one that matches your intent.
{: .fs-5 .fw-300 }

---

## At-a-Glance

```mermaid
flowchart TB
    subgraph "Lifecycle (5)"
        A[ticket_create]
        B[ticket_get]
        C[ticket_claim]
        D[ticket_update]
        E[ticket_archive]
    end
    subgraph "Discovery (6)"
        F[ticket_list]
        G[ticket_backlog]
        H[ticket_board]
        I[ticket_search]
        J[ticket_children]
        K[ticket_diagram]
    end
    subgraph "Collaboration (2)"
        L[ticket_comment]
        M[ticket_link]
    end
```

---

## 🔵 Lifecycle (5)

| Tool | Required | Returns | When |
|------|----------|---------|------|
| `ticket_create` | `title` | `T{...}` | Adding new work |
| `ticket_get` | `id` | `T{...}` (with `cmt`, `lnk`, `ch`) | Reading a single ticket |
| `ticket_claim` | `id` | `T{s:ip,...}` (atomic) | Taking ownership |
| `ticket_update` | `id` | `T{...}` | Changing any field (always send `etag`) |
| `ticket_archive` | `id` | `{ok:true}` | Hiding finished work |

---

## 🟢 Discovery (6)

| Tool | Returns | Best For |
|------|---------|----------|
| `ticket_list` | `[T{...}]` | Filtered queries (`status`, `type`, `labels`) |
| `ticket_backlog` | `[T{...}]` priority-ordered | **Start here.** "What should I work on?" |
| `ticket_board` | `BOARD{bk:[],ip:[],...}` | Kanban snapshot |
| `ticket_search` | `[T{...}]` ranked | FTS5 keyword search |
| `ticket_children` | `[T{...}]` | Listing subtasks of a parent |
| `ticket_diagram` | Mermaid flowchart | Visualizing a hierarchy |

---

## 🟣 Collaboration (2)

| Tool | Required | Returns |
|------|----------|---------|
| `ticket_comment` | `id`, `body` | `T{...}` (updated ticket) |
| `ticket_link` | `from_id`, `to_id`, `link_type` | `{ok:true}` |

`link_type` is `blk` (blocks), `rel` (relates), or `dup` (duplicates).

---

## 📚 Deeper Reference

- **[API Guide]({{ site.baseurl }}/tools/api-guide)** — every parameter, every default
- **[Examples]({{ site.baseurl }}/tools/examples)** — annotated end-to-end workflows
- **[Troubleshooting]({{ site.baseurl }}/tools/troubleshooting)** — error code recovery playbook
