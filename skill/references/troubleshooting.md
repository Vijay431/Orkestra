# Orkestra — Troubleshooting & Error Recovery

---

## Error Code Reference

### `not_found`

**Cause:** Ticket ID doesn't exist, was archived, or belongs to a different project.

**Symptoms:**
```
TOON/1 ERR{code:not_found,msg:"myapp-999 does not exist"}
```

**Recovery:**
1. Double-check the ticket ID (format: `{PROJECT_ID}-{NNN}`)
2. Use `ticket_search query="<keyword>"` to locate the ticket by text
3. If the ticket might be archived, use `ticket_list include_archived=true` or `ticket_search include_archived=true`
4. Verify the MCP server's `PROJECT_ID` matches the ticket prefix

---

### `conflict`

**Cause A — Stale etag on `ticket_update`:**
Another agent (or the same agent in a parallel operation) updated the ticket between your last read and your update attempt.

**Cause B — Already claimed on `ticket_claim`:**
The ticket is already in `ip`, `dn`, or `cl` status.

**Symptoms:**
```
TOON/1 ERR{code:conflict,msg:"etag mismatch for myapp-003"}
TOON/1 ERR{code:conflict,msg:"myapp-003 already claimed"}
```

**Recovery — etag conflict:**
```
# Re-read the ticket to get current state and fresh etag
ticket_get id=myapp-003
→ T{...,ua:2024-01-15T10:08:45.987654321Z}

# Retry with fresh etag
ticket_update id=myapp-003 etag=2024-01-15T10:08:45.987654321Z s=dn
```

**Recovery — already claimed:**
```
# Check current state
ticket_get id=myapp-003
→ T{s:ip,...}  # being worked on by another agent

# If the ticket is stuck in-progress with no recent activity, you can reclaim it
ticket_update id=myapp-003 etag=<ua> s=bk   # reset to backlog
ticket_claim id=myapp-003                    # reclaim
```

---

### `seq_blocked`

**Cause:** You tried to claim a sequential child ticket whose predecessor (`exec_order N-1`) is not yet done.

**Symptoms:**
```
TOON/1 ERR{code:seq_blocked,msg:"myapp-022 blocked: ord=1 not done"}
```

**Recovery:**
```
# Find the parent and inspect siblings
ticket_get id=myapp-022
→ T{par:myapp-020,ord:2,...}

ticket_children id=myapp-020
→ [
    T{id:myapp-021,t:Run tests,ord:1,s:bk,...},   # ← this one is not done
    T{id:myapp-022,t:Build image,ord:2,s:bk,...},
    T{id:myapp-023,t:Push to registry,ord:3,s:bk,...}
  ]

# Complete the predecessor first
ticket_claim id=myapp-021
(do the work)
ticket_update id=myapp-021 s=dn etag=<ua>

# Now you can claim the next step
ticket_claim id=myapp-022  →  T{s:ip,...}  ✓
```

---

### `invalid`

**Cause:** Bad input — most commonly a duplicate `exec_order` value within the same parent.

**Symptoms:**
```
TOON/1 ERR{code:invalid,msg:"exec_order must be unique within parent"}
```

**Recovery:**
```
# Check existing exec_order values among siblings
ticket_children id=<parent_id>
→ [T{ord:1,...}, T{ord:2,...}, T{ord:3,...}]

# Use the next available exec_order
ticket_create title="New step" parent_id=<parent_id> exec_mode=seq exec_order=4
```

---

### `internal`

**Cause:** Unexpected server-side error (database issue, write failure, etc.).

**Symptoms:**
```
TOON/1 ERR{code:internal,msg:"..."}
```

**Recovery:**
1. Retry the same operation once — transient SQLite write conflicts resolve quickly
2. Check server health: `curl http://localhost:8080/health`
   - `db_ok: false` → database file may be locked or corrupted
3. Check server logs: `docker compose logs -f`
4. If the problem persists, restart the container: `docker compose restart`

---

## Common Issues

### "I called ticket_backlog but got an empty list"

**Cause:** All tickets are either in `ip`/`dn`/`cl` status, or the backlog is genuinely empty.

**Diagnosis:**
```
ticket_board  # check all statuses
→ BOARD{ip:[T{...}],dn:[T{...}]}
# → confirm there are tickets but none in bk status
```

If everything is in progress, check if previous agents got stuck:
```
ticket_list status=ip
# Look for tickets that have been in_progress for a long time
# with no recent updates — may need to be reset to bk
```

---

### "ticket_update keeps getting ERR{code:conflict}"

**Cause:** High-concurrency situation with many agents updating the same ticket.

**Pattern — exponential backoff:**
```
# Attempt 1
ticket_update id=X etag=<ua1> s=dn  →  ERR{code:conflict}

# Re-read
ticket_get id=X  →  T{...,ua:<ua2>}

# Attempt 2
ticket_update id=X etag=<ua2> s=dn  →  ERR{code:conflict} (still racing)

# Re-read again
ticket_get id=X  →  T{s:dn,...}
# → another agent already marked it done — no action needed
```

---

### "ticket_claim succeeded but the ticket shows stale data in later reads"

**Cause:** Not a bug — SQLite WAL mode can briefly show stale reads in concurrent scenarios. The data will be consistent on the next read.

**Fix:** If you need the latest state immediately after a write, call `ticket_get` to confirm.

---

### "The server health endpoint returns `db_ok: false`"

**Cause:** SQLite file is locked (another process holds a write lock) or the database path is wrong.

**Fix:**
```bash
# Check if another process is writing
lsof /data/orkestra.db

# Verify the DB_PATH environment variable
docker compose exec orkestra env | grep DB_PATH

# Restart to release any stale locks
docker compose restart
```

---

### "Sequential pipeline children aren't showing exec_order"

**Cause:** The parent was created with `exec_mode=par` (default) and children were created without `exec_order`.

**Diagnosis:**
```
ticket_get id=<parent>
→ T{em:par,...}  # ← parent is parallel, not sequential
```

**Fix:** Either create a new parent with `exec_mode=seq` and recreate the children with `exec_order`, or update the parent:
```
ticket_update id=<parent> exec_mode=seq etag=<ua>
# Then update each child to add exec_order
ticket_update id=<child1> exec_order=1 etag=<ua>
ticket_update id=<child2> exec_order=2 etag=<ua>
```
