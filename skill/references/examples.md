# Orkestra — Workflow Examples

Annotated end-to-end examples with realistic TOON responses.

---

## 1. Basic Work Loop

Pick the highest-priority ticket, claim it, do the work, mark it done.

```
# Step 1: find highest-priority work
ticket_backlog
→ TOON/1 [
    T{id:myapp-003,t:Fix session expiry,s:bk,p:h,typ:bug,lbl:[auth],ca:2024-01-15,ua:2024-01-15T09:00:00Z},
    T{id:myapp-004,t:Add login rate limit,s:bk,p:m,typ:ft,ca:2024-01-14,ua:2024-01-14T08:00:00Z}
  ]
# → pick myapp-003 (higher priority: h vs m)

# Step 2: claim it atomically
ticket_claim id=myapp-003
→ TOON/1 T{id:myapp-003,t:Fix session expiry,s:ip,p:h,typ:bug,lbl:[auth],ca:2024-01-15,ua:2024-01-15T10:05:22.123456789Z}
#                                                                                          ↑ save this as etag

# Step 3: add a progress note
ticket_comment id=myapp-003 body="Root cause identified: session TTL not reset on activity"
→ TOON/1 T{id:myapp-003,...,cmt:[C{a:llm,t:"Root cause identified: session TTL not reset on activity",ts:2024-01-15T10:10}]}

# Step 4: mark done (supply etag)
ticket_update id=myapp-003 s=dn etag=2024-01-15T10:05:22.123456789Z
→ TOON/1 T{id:myapp-003,t:Fix session expiry,s:dn,p:h,typ:bug,ca:2024-01-15,ua:2024-01-15T10:15:00.000000000Z}
```

---

## 2. Epic with Parallel Swarm

Multiple agents can claim different children simultaneously.

```
# Create the epic
ticket_create title="Auth system" type=ep
→ TOON/1 T{id:myapp-010,t:Auth system,s:bk,typ:ep,ca:2024-01-15,ua:2024-01-15T11:00:00Z}

# Create 3 parallel children (exec_mode=par is the default)
ticket_create title="JWT middleware"  parent_id=myapp-010
→ TOON/1 T{id:myapp-011,t:JWT middleware,s:bk,p:m,typ:tsk,par:myapp-010,ca:2024-01-15,...}

ticket_create title="OAuth provider"  parent_id=myapp-010
→ TOON/1 T{id:myapp-012,t:OAuth provider,s:bk,p:m,typ:tsk,par:myapp-010,ca:2024-01-15,...}

ticket_create title="Session store"   parent_id=myapp-010
→ TOON/1 T{id:myapp-013,t:Session store,s:bk,p:m,typ:tsk,par:myapp-010,ca:2024-01-15,...}

# Visualize
ticket_diagram id=myapp-010
→ ```mermaid
  flowchart TD
    myapp_010["Auth system [ep|bk|m]"]
    subgraph par_myapp_010["⚡ Parallel"]
      myapp_011["JWT middleware [tsk|bk|m]"]
      myapp_012["OAuth provider [tsk|bk|m]"]
      myapp_013["Session store [tsk|bk|m]"]
    end
    myapp_010 --> par_myapp_010
  ```

# Agent A claims JWT middleware; Agent B simultaneously claims OAuth provider
# Both succeed because exec_mode=par allows concurrent claims
ticket_claim id=myapp-011  →  T{s:ip,...}   # Agent A
ticket_claim id=myapp-012  →  T{s:ip,...}   # Agent B (concurrent, no conflict)
```

---

## 3. Sequential Pipeline

Steps must run in order — later steps are blocked until predecessors are done.

```
# Create the pipeline parent
ticket_create title="Deploy pipeline" type=tsk exec_mode=seq
→ TOON/1 T{id:myapp-020,t:Deploy pipeline,s:bk,typ:tsk,em:seq,ca:2024-01-15,...}

# Create steps in order
ticket_create title="Run tests"        parent_id=myapp-020 exec_mode=seq exec_order=1
→ TOON/1 T{id:myapp-021,...,em:seq,ord:1,par:myapp-020,...}

ticket_create title="Build image"      parent_id=myapp-020 exec_mode=seq exec_order=2
→ TOON/1 T{id:myapp-022,...,em:seq,ord:2,par:myapp-020,...}

ticket_create title="Push to registry" parent_id=myapp-020 exec_mode=seq exec_order=3
→ TOON/1 T{id:myapp-023,...,em:seq,ord:3,par:myapp-020,...}

# Attempting to claim step 2 before step 1 is done:
ticket_claim id=myapp-022
→ TOON/1 ERR{code:seq_blocked,msg:"myapp-022 blocked: ord=1 not done"}
# → must claim and complete myapp-021 (ord=1) first

ticket_claim id=myapp-021
→ T{id:myapp-021,s:ip,...,ua:2024-01-15T12:05:00.123Z}

ticket_update id=myapp-021 s=dn etag=2024-01-15T12:05:00.123Z
→ T{id:myapp-021,s:dn,...}

# Now step 2 can be claimed
ticket_claim id=myapp-022
→ T{id:myapp-022,s:ip,...}  ✓
```

---

## 4. Concurrent Agent Conflict (Etag Retry)

Two agents try to update the same ticket — one wins, the other retries safely.

```
# Agent A and Agent B both read myapp-003 (ua = 2024-01-15T10:05:22.123Z)

# Agent A updates first — succeeds
ticket_update id=myapp-003 s=dn etag=2024-01-15T10:05:22.123456789Z
→ TOON/1 T{id:myapp-003,s:dn,...,ua:2024-01-15T10:08:00.000000000Z}

# Agent B tries with stale etag — fails
ticket_update id=myapp-003 s=bl etag=2024-01-15T10:05:22.123456789Z
→ TOON/1 ERR{code:conflict,msg:"etag mismatch for myapp-003"}

# Agent B re-reads to get fresh state and etag
ticket_get id=myapp-003
→ TOON/1 T{id:myapp-003,s:dn,...,ua:2024-01-15T10:08:00.000000000Z}
# → ticket is already done; Agent B decides no update needed
```

---

## 5. Multi-Agent Backlog Drain

Multiple agents independently picking up work without coordination.

```
# All agents call ticket_backlog — they all see the same list
# Agent A claims myapp-003
ticket_claim id=myapp-003  →  T{s:ip,...}  ✓

# Agent B also tries myapp-003 — lost the race
ticket_claim id=myapp-003  →  ERR{code:conflict,msg:"myapp-003 already claimed"}
# → Agent B moves to the next item
ticket_claim id=myapp-004  →  T{s:ip,...}  ✓

# Agent C claims myapp-005 simultaneously with no conflict
ticket_claim id=myapp-005  →  T{s:ip,...}  ✓
```

---

## 6. Searching and Linking Related Tickets

```
# Find all auth-related tickets
ticket_search query="auth login session"
→ TOON/1 [
    T{id:myapp-003,t:Fix session expiry,s:dn,...},
    T{id:myapp-011,t:JWT middleware,s:ip,...},
    T{id:myapp-015,t:Auth regression on mobile,s:bk,...}
  ]

# Link the regression to the original fix (myapp-003 blocks myapp-015)
ticket_link from_id=myapp-003 to_id=myapp-015 link_type=rel
→ TOON/1 {ok:true}

# Mark myapp-015 as blocked by myapp-003
ticket_link from_id=myapp-003 to_id=myapp-015 link_type=blk
→ TOON/1 {ok:true}

# Verify links appear on ticket_get
ticket_get id=myapp-015
→ TOON/1 T{id:myapp-015,...,lnk:[L{f:myapp-003,t:myapp-015,k:blk},L{f:myapp-003,t:myapp-015,k:rel}]}
```
