# Orkestra — Full Tool Parameter Reference

All 13 tools with complete parameter details. For quick reference see `SKILL.md`.

---

## ticket_create

Create a new ticket. Always lands in backlog (`s=bk`).

| Param | Required | Type | Default | Valid values / Notes |
|-------|----------|------|---------|----------------------|
| `title` | **yes** | string | — | Non-empty |
| `type` | no | enum | `tsk` | `bug` `ft` `tsk` `ep` `chr` |
| `priority` | no | enum | `m` | `cr` `h` `m` `l` |
| `description` | no | string | — | |
| `labels` | no | string[] | [] | |
| `parent_id` | no | string | — | ID of parent ticket |
| `exec_mode` | no | enum | `par` | `par` `seq` |
| `exec_order` | no | integer | — | Required for `seq` children; `UNIQUE(parent_id, exec_order)` at DB level |

**Returns:** `TOON/1 T{id:myapp-011,t:Fix login bug,s:bk,p:h,typ:bug,ca:2024-01-15,ua:2024-01-15T10:00:00Z}`

---

## ticket_get

Fetch a single ticket with full relations (comments, links, child IDs).

| Param | Required |
|-------|----------|
| `id` | **yes** |

**Returns:** Complete `T{...}` including `cmt`, `lnk`, `ch` arrays.
**Errors:** `ERR{code:not_found}` if ticket doesn't exist or is archived.

---

## ticket_claim

Atomically take ownership. Moves ticket from `bk`/`td`/`bl` → `ip`.

| Param | Required |
|-------|----------|
| `id` | **yes** |

**Returns:** `T{s:ip,...}` — **save `ua` as your etag for `ticket_update`**.

| Error | Meaning | Action |
|-------|---------|--------|
| `ERR{code:conflict}` | Already claimed/done/cancelled | Move to next backlog item |
| `ERR{code:seq_blocked,msg:"ord=N predecessor not done"}` | Earlier step not finished | Complete lower `ord` sibling first |
| `ERR{code:not_found}` | Wrong ID or archived | Verify with `ticket_search` |

---

## ticket_update

Update any field on a ticket. Always supply `etag` when working with concurrent agents.

| Param | Required | Notes |
|-------|----------|-------|
| `id` | **yes** | |
| `etag` | recommended | The `ua` value from your last read. Appends `WHERE updated_at = ?` — stale etag → `ERR{code:conflict}` |
| `title` | no | |
| `status` | no | `bk` `td` `ip` `dn` `bl` `cl` |
| `priority` | no | `cr` `h` `m` `l` |
| `type` | no | `bug` `ft` `tsk` `ep` `chr` |
| `description` | no | |
| `assignee` | no | |
| `labels` | no | Replaces **entire** label array when provided |
| `exec_mode` | no | `par` `seq` |
| `exec_order` | no | Must be unique within parent |

**Returns:** Updated `T{...}`.
`ERR{code:conflict}` → etag stale; call `ticket_get`, retry with fresh `ua`.

---

## ticket_archive

Soft-delete. Sets `archived_at`. Ticket excluded from all queries.

| Param | Required |
|-------|----------|
| `id` | **yes** |

**Returns:** `{ok:true}`. No unarchive tool.

---

## ticket_list

Filter tickets across any combination of fields. Ordered by `created_at DESC`.

| Param | Type | Default | Notes |
|-------|------|---------|-------|
| `status` | enum | — | `bk` `td` `ip` `dn` `bl` `cl` |
| `priority` | enum | — | `cr` `h` `m` `l` |
| `type` | enum | — | `bug` `ft` `tsk` `ep` `chr` |
| `labels` | string[] | — | AND filter — all labels must match |
| `limit` | integer | 50 | Max 200 |
| `include_archived` | boolean | false | |

**Returns:** `TOON/1 [T{...},...]`

---

## ticket_backlog

Priority-ordered view of `bk` tickets. Use this to decide what to do next.

| Param | Type | Notes |
|-------|------|-------|
| `priority` | enum | Optional filter |
| `type` | enum | Optional filter |
| `labels` | string[] | Optional filter |
| `limit` | integer | Default 50 |

**Returns:** `TOON/1 [...]` — ordered `cr → h → m → l`, then oldest first within each priority.

---

## ticket_board

Kanban snapshot grouped by status.

| Param | Type | Notes |
|-------|------|-------|
| `type` | enum | Optional filter |
| `labels` | string[] | Optional filter |

**Returns:** `TOON/1 BOARD{bk:[...],td:[...],ip:[...],dn:[...],bl:[...],cl:[...]}` — empty buckets omitted. Status order: `bk, td, ip, bl, cl, dn`.

---

## ticket_search

Full-text search across title, description, and labels using SQLite FTS5.

| Param | Required | Notes |
|-------|----------|-------|
| `query` | **yes** | FTS5 ranked — supports phrase search |
| `include_archived` | no | Default false |

**Returns:** `TOON/1 [...]` ranked by relevance.

---

## ticket_children

List children of a ticket, sorted by `exec_order`.

| Param | Type | Default | Notes |
|-------|------|---------|-------|
| `id` | string | — | **required** |
| `recursive` | boolean | false | If true, returns full subtree up to `depth` |
| `depth` | integer | 3 | 1–10; only used when `recursive=true` |

**Returns:** `TOON/1 [...]`. Sequential children in `exec_order` order; parallel children in `created_at` order.

---

## ticket_diagram

Mermaid flowchart of ticket hierarchy with status colors.

| Param | Type | Default | Notes |
|-------|------|---------|-------|
| `id` | string | — | **required** — root of the diagram |
| `depth` | integer | 3 | 1–10 |

**Returns:** Mermaid `flowchart TD` code block.
- Parallel children → `⚡ Parallel` subgraph
- Sequential children → `🔗 Sequential` subgraph with order arrows (1→2→3)
- Status colors: `#aaaaaa` gray (bk) · `#f0c040` yellow (td) · `#40a0f0` blue (ip) · `#40c040` green (dn) · `#f04040` red (bl) · `#cccccc` light-gray (cl)

---

## ticket_comment

Append a note to a ticket.

| Param | Required | Notes |
|-------|----------|-------|
| `id` | **yes** | |
| `body` | **yes** | Comment text |
| `author` | no | Defaults to `llm` |

**Returns:** Updated ticket with new comment appended to `cmt` array.
Comment encoding: `C{a:author,t:"body",ts:2024-01-15T10:04}`

---

## ticket_link

Create a directional relationship between two tickets.

| Param | Required | Values |
|-------|----------|--------|
| `from_id` | **yes** | Source ticket |
| `to_id` | **yes** | Target ticket |
| `link_type` | **yes** | `blk` (from blocks to) · `rel` (relates) · `dup` (duplicates) |

**Returns:** `{ok:true}`. Idempotent (INSERT OR IGNORE).
Link encoding in responses: `L{f:from_id,t:to_id,k:blk}`
