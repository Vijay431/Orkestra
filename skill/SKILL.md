---
name: orkestra
version: 1.0.0
description: |
  Local MCP ticket server for planning, tracking, and coordinating work across
  swarming agents. Provides 13 tools for creating, claiming, updating, searching,
  and visualizing tickets — all stored in local SQLite with no cloud dependency.
  Use ticket_backlog to find work, ticket_claim to take ownership, ticket_update
  to mark done. See references/ for full tool parameters, examples, and troubleshooting.
allowed-tools:
  - Read
---

# Orkestra — LLM Operator Guide

## What You Can Do

Orkestra is a **local ticket server** for planning, tracking, and coordinating work across agents. No cloud, no rate limits. All data in SQLite.

| Intent | Tool | Notes |
|--------|------|-------|
| What should I work on next? | `ticket_backlog` | Priority-ordered (cr→h→m→l). **Start here.** |
| See all work across statuses | `ticket_board` | Kanban snapshot grouped by status |
| Find a ticket by keyword | `ticket_search` | FTS5 full-text search |
| Filter by status / type / label | `ticket_list` | Use after backlog gives too many results |
| Read a ticket (with comments + links) | `ticket_get` | Full detail including `cmt`, `lnk`, `ch` |
| Take ownership of a ticket | `ticket_claim` | Atomic — fails safely if already claimed |
| Mark done / change any field | `ticket_update` | Always supply `etag=<ua>` |
| Create a task | `ticket_create` | Lands in backlog; `parent_id` for subtasks |
| Break work into subtasks | `ticket_create` (loop) | `exec_mode=par` (swarm) or `seq` (pipeline) |
| List subtasks | `ticket_children` | Sorted by `exec_order` |
| Visualize hierarchy | `ticket_diagram` | Returns Mermaid flowchart |
| Add a progress note | `ticket_comment` | |
| Link two tickets | `ticket_link` | `blk` blocks · `rel` relates · `dup` duplicates |
| Hide a finished ticket | `ticket_archive` | Soft-delete |

**Do not** use `ticket_list` to find what to work on — use `ticket_backlog` which applies priority ordering automatically.

---

## Reading TOON Responses

All responses start with `TOON/1`. Fields use short aliases:

| TOON  | Meaning      | Values / Notes |
|-------|--------------|----------------|
| `id`  | ticket ID    | `{PROJECT_ID}-{NNN}` |
| `t`   | title        | string |
| `s`   | status       | `bk` backlog · `td` todo · `ip` in_progress · `dn` done · `bl` blocked · `cl` cancelled |
| `p`   | priority     | `cr` critical · `h` high · `m` medium · `l` low |
| `typ` | type         | `bug` · `ft` feature · `tsk` task · `ep` epic · `chr` chore |
| `em`  | exec_mode    | `par` parallel (default, **omitted**) · `seq` sequential |
| `ord` | exec_order   | integer — only for sequential children |
| `lbl` | labels       | `["tag1","tag2"]` — omitted if empty |
| `par` | parent_id    | omitted if root ticket |
| `ch`  | children IDs | call `ticket_get` or `ticket_children` to expand |
| `d`   | description  | omitted if empty |
| `as`  | assignee     | omitted if unassigned |
| `ua`  | updated_at   | RFC3339Nano — **use as `etag` in `ticket_update`** |
| `ca`  | created_at   | date |
| `cmt` | comments     | `C{a:author,t:"body",ts:timestamp}` — only in `ticket_get` |
| `lnk` | links        | `L{f:from,t:to,k:blk\|rel\|dup}` — only in `ticket_get` |

**Board:** `TOON/1 BOARD{bk:[...],td:[...],ip:[...],dn:[...],bl:[...],cl:[...]}` — empty buckets omitted
**Error:** `TOON/1 ERR{code:not_found,msg:"myapp-999 does not exist"}`
**Success:** `TOON/1 {ok:true}`

---

## Core Workflow

```
ticket_backlog
→ TOON/1 [T{id:myapp-003,t:Fix session expiry,s:bk,p:h,typ:bug,ca:2024-01-15,ua:2024-01-15T09:00:00Z},...]

ticket_claim id=myapp-003
→ TOON/1 T{id:myapp-003,s:ip,...,ua:2024-01-15T10:05:22.123456789Z}
#                                  ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^ save as etag

(do the work)

ticket_update id=myapp-003 s=dn etag=2024-01-15T10:05:22.123456789Z
→ TOON/1 T{id:myapp-003,s:dn,...}
```

**Etag conflict:** If you get `ERR{code:conflict}` on `ticket_update`, another agent updated first. Call `ticket_get` to get the fresh `ua`, then retry.

**Sequential claim blocked:** If you get `ERR{code:seq_blocked}`, the previous `exec_order` step is not done yet. Call `ticket_children` on the parent to find and complete the predecessor.

---

## Tool Quick Reference

| Tool | Required | Key Optional | Returns |
|------|----------|-------------|---------|
| `ticket_create` | `title` | `type` `priority` `description` `labels` `parent_id` `exec_mode` `exec_order` | `T{...}` |
| `ticket_get` | `id` | — | `T{...}` with `cmt` `lnk` `ch` |
| `ticket_claim` | `id` | — | `T{s:ip,...}` |
| `ticket_update` | `id` | `etag` `status` `priority` `type` `title` `assignee` `labels` | `T{...}` |
| `ticket_archive` | `id` | — | `{ok:true}` |
| `ticket_list` | — | `status` `priority` `type` `labels` `limit`(50) `include_archived` | `[T{...}]` |
| `ticket_backlog` | — | `priority` `type` `labels` `limit`(50) | `[T{...}]` priority-ordered |
| `ticket_board` | — | `type` `labels` | `BOARD{...}` |
| `ticket_search` | `query` | `include_archived` | `[T{...}]` ranked |
| `ticket_children` | `id` | `recursive`(false) `depth`(3) | `[T{...}]` |
| `ticket_diagram` | `id` | `depth`(3) | Mermaid chart |
| `ticket_comment` | `id` `body` | `author` | `T{...}` |
| `ticket_link` | `from_id` `to_id` `link_type` | — | `{ok:true}` |

**Valid enums:**
- `status`: `bk` `td` `ip` `dn` `bl` `cl`
- `priority`: `cr` `h` `m` `l`
- `type`: `bug` `ft` `tsk` `ep` `chr`
- `exec_mode`: `par` `seq`
- `link_type`: `blk` `rel` `dup`

---

## Error Codes

| Code | Cause | Action |
|------|-------|--------|
| `not_found` | Ticket doesn't exist or is archived | Verify ID; use `ticket_search` |
| `conflict` | Etag stale or ticket already claimed | `ticket_get` → retry with fresh `ua` |
| `seq_blocked` | Sequential predecessor not done | Complete lower `exec_order` sibling first |
| `invalid` | Duplicate `exec_order` in same parent | `ticket_children` → choose unique `ord` |
| `internal` | Server error | Retry once; check `/health` if it persists |

---

> **Full detail:** See `references/api-guide.md` for all tool parameters, `references/examples.md` for annotated workflows, `references/troubleshooting.md` for error recovery.
