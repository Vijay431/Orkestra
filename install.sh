#!/usr/bin/env bash
set -euo pipefail

# Orkestra install script — onboards Orkestra MCP server into AI tool configs.
#
# Usage (remote — no clone required):
#   curl -fsSL https://raw.githubusercontent.com/Vijay431/Orkestra/main/install.sh | PROJECT_ID=myapp bash
#
# Usage (local clone):
#   git clone https://github.com/Vijay431/Orkestra && cd Orkestra
#   PROJECT_ID=myapp ./install.sh
#
# Or run interactively (prompts for inputs when env vars are unset).

# ---------- helpers ----------

info()  { printf '\033[32m[orkestra]\033[0m %s\n' "$*"; }
warn()  { printf '\033[33m[orkestra]\033[0m %s\n' "$*"; }
error() { printf '\033[31m[orkestra]\033[0m %s\n' "$*" >&2; }
die()   { error "$*"; exit 1; }

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "Required command not found: $1"
}

# ---------- preflight ----------

require_cmd docker
docker compose version >/dev/null 2>&1 || die "docker compose plugin is required"

# ---------- bootstrap (remote install support) ----------
# When run via `curl … | bash`, BASH_SOURCE[0] is empty or "bash" / "/dev/stdin",
# so there is no local repo to draw Dockerfile/skill/ from.
# Detect this and clone (or update) the repo to ORKESTRA_HOME first.

ORKESTRA_REPO="https://github.com/Vijay431/Orkestra"
ORKESTRA_HOME="${ORKESTRA_HOME:-$HOME/.orkestra}"

# Resolve the directory that contains this script (empty when piped).
_raw_source="${BASH_SOURCE[0]:-}"
if [[ -n "$_raw_source" && "$_raw_source" != "bash" && "$_raw_source" != "/dev/stdin" ]]; then
  SCRIPT_DIR="$(cd "$(dirname "$_raw_source")" && pwd)"
else
  SCRIPT_DIR=""
fi

# If SCRIPT_DIR is empty or the resolved dir has no Dockerfile, we need to bootstrap.
if [[ -z "$SCRIPT_DIR" || ! -f "$SCRIPT_DIR/Dockerfile" ]]; then
  require_cmd git
  if [[ -d "$ORKESTRA_HOME/.git" ]]; then
    info "Updating existing Orkestra clone at $ORKESTRA_HOME ..."
    git -C "$ORKESTRA_HOME" pull --ff-only --quiet
  else
    info "Cloning Orkestra into $ORKESTRA_HOME ..."
    git clone --depth 1 --quiet "$ORKESTRA_REPO" "$ORKESTRA_HOME"
  fi
  SCRIPT_DIR="$ORKESTRA_HOME"
  info "Using repo at $SCRIPT_DIR"
fi

# ---------- inputs ----------

if [[ -z "${PROJECT_ID:-}" ]]; then
  read -rp "Project ID (e.g. auth, payments, myapp): " PROJECT_ID
fi
[[ -n "$PROJECT_ID" ]] || die "PROJECT_ID is required"

PORT="${PORT:-8080}"
if [[ -z "${PORT_SET:-}" ]]; then
  read -rp "Port [${PORT}]: " input_port
  PORT="${input_port:-$PORT}"
fi

TARGET_DIR="${TARGET_DIR:-$(pwd)}"

# Generate MCP token if not provided
MCP_TOKEN="${MCP_TOKEN:-}"
if [[ -z "$MCP_TOKEN" ]]; then
  if command -v openssl >/dev/null 2>&1; then
    MCP_TOKEN=$(openssl rand -hex 32)
    info "Generated MCP_TOKEN (stored in .env)"
  fi
fi

# ---------- write docker-compose.yml ----------

COMPOSE_FILE="$TARGET_DIR/docker-compose.yml"
if [[ ! -f "$COMPOSE_FILE" ]]; then
  if [[ -f "$SCRIPT_DIR/docker-compose.yml" ]]; then
    cp "$SCRIPT_DIR/docker-compose.yml" "$COMPOSE_FILE"
    info "Copied docker-compose.yml to $TARGET_DIR"
  fi
fi

# Write .env with the project-specific values
ENV_FILE="$TARGET_DIR/.env"
cat > "$ENV_FILE" <<EOF
PROJECT_ID=${PROJECT_ID}
PORT=${PORT}
MCP_TOKEN=${MCP_TOKEN}
BIND_ADDR=0.0.0.0
EOF
info "Written $ENV_FILE"

# ---------- start container ----------

COMPOSE_FLAGS="-f $COMPOSE_FILE --env-file $ENV_FILE"

if [[ -f "$SCRIPT_DIR/Dockerfile" ]]; then
  info "Building Docker image..."
  docker compose $COMPOSE_FLAGS build
else
  info "Pulling Docker image..."
  docker compose $COMPOSE_FLAGS pull 2>/dev/null || true
fi

info "Starting Orkestra container..."
docker compose $COMPOSE_FLAGS up -d

# ---------- wait for health ----------

BASE_URL="http://localhost:${PORT}"
info "Waiting for health check at ${BASE_URL}/health ..."
DEADLINE=$(($(date +%s) + 30))
until curl -sf "${BASE_URL}/health" >/dev/null 2>&1; do
  [[ $(date +%s) -lt $DEADLINE ]] || die "Orkestra did not become healthy within 30s"
  sleep 1
done
info "Orkestra is healthy"

# ---------- install skill globally ----------
# The Orkestra skill lives in skill/ (SKILL.md + references/).
# It is installed to ~/.agents/skills/orkestra/ — the central hub that all
# AI tools symlink into their own skills/ directories.

SKILL_DIR="$SCRIPT_DIR/skill"
SKILL_SRC="$SKILL_DIR/SKILL.md"              # used by Claude Code CLAUDE.md injection
GLOBAL_SKILLS_HUB="${HOME}/.agents/skills"
SKILL_INSTALL_DIR="${GLOBAL_SKILLS_HUB}/orkestra"

install_skill_globally() {
  if [[ ! -d "$SKILL_DIR" ]]; then
    warn "skill/ directory not found — skipping global skill installation"
    return
  fi

  # Install skill package to central hub
  mkdir -p "$SKILL_INSTALL_DIR"
  # Use rsync if available for clean sync; fall back to cp
  if command -v rsync >/dev/null 2>&1; then
    rsync -a --delete "$SKILL_DIR/" "$SKILL_INSTALL_DIR/"
  else
    cp -r "$SKILL_DIR/." "$SKILL_INSTALL_DIR/"
  fi
  info "Installed skill to $SKILL_INSTALL_DIR"

  # Create symlinks in each AI tool's skills/ directory
  local tools_skills=(
    "${HOME}/.claude/skills"
    "${HOME}/.cursor/skills"
    "${HOME}/.codeium/windsurf/skills"
    "${HOME}/.continue/skills"
  )
  for tool_skills_dir in "${tools_skills[@]}"; do
    if [[ -d "$tool_skills_dir" ]]; then
      local link="$tool_skills_dir/orkestra"
      # Relative path from tool_skills_dir to SKILL_INSTALL_DIR
      # Both are under $HOME so compute relative path with Python
      local rel_target
      rel_target=$(python3 -c "
import os
link_dir = '$tool_skills_dir'
target   = '$SKILL_INSTALL_DIR'
print(os.path.relpath(target, link_dir))
" 2>/dev/null) || rel_target="$SKILL_INSTALL_DIR"

      if [[ -L "$link" ]]; then
        # Update existing symlink
        ln -sf "$rel_target" "$link"
      elif [[ ! -e "$link" ]]; then
        ln -s "$rel_target" "$link"
      fi
      info "Linked skill in $tool_skills_dir"
    fi
  done
}

install_skill_globally

# ---------- detect and register AI tools ----------

SSE_URL="${BASE_URL}/sse"
MCP_NAME="orkestra-${PROJECT_ID}"
AUTH_HEADER=""
[[ -n "$MCP_TOKEN" ]] && AUTH_HEADER="Authorization: Bearer ${MCP_TOKEN}"

registered=()
skipped=()

## Claude Code ##
if command -v claude >/dev/null 2>&1; then
  info "Registering with Claude Code..."
  claude mcp add "$MCP_NAME" --transport http "$SSE_URL" 2>/dev/null || true

  # Inject skill content into ~/.claude/CLAUDE.md so the skill is active in every
  # Claude Code conversation, not just sessions started from this project directory.
  # Uses HTML comment sentinels for idempotent updates (re-running replaces the block).
  # Writes atomically via temp file. Source: skill/SKILL.md (canonical skill file).
  GLOBAL_CLAUDE="${HOME}/.claude/CLAUDE.md"
  if [[ -f "$SKILL_SRC" ]]; then
    # Backup before modifying
    [[ -f "$GLOBAL_CLAUDE" ]] && cp "$GLOBAL_CLAUDE" "${GLOBAL_CLAUDE}.bak"

    python3 - "$GLOBAL_CLAUDE" "$SKILL_SRC" <<'PYEOF' 2>/dev/null || warn "Could not update ~/.claude/CLAUDE.md — add ORKESTRA_SKILL.md content manually"
import sys, os, tempfile

target   = sys.argv[1]
skill_src = sys.argv[2]

with open(skill_src) as f:
    skill_content = f.read().strip()

begin_marker = "<!-- orkestra-skill-begin -->"
end_marker   = "<!-- orkestra-skill-end -->"
skill_block  = f"{begin_marker}\n{skill_content}\n{end_marker}"

if os.path.exists(target):
    with open(target) as f:
        existing = f.read()
    if begin_marker in existing and end_marker in existing:
        # Replace existing block between markers
        before = existing[:existing.index(begin_marker)]
        after  = existing[existing.index(end_marker) + len(end_marker):]
        new_content = before.rstrip('\n') + '\n\n' + skill_block + '\n' + after.lstrip('\n')
    else:
        # Append block to end of existing file
        new_content = existing.rstrip('\n') + '\n\n' + skill_block + '\n'
else:
    new_content = skill_block + '\n'

# Atomic write: write to temp file, then rename into place
target_dir = os.path.dirname(os.path.abspath(target))
os.makedirs(target_dir, exist_ok=True)
with tempfile.NamedTemporaryFile(mode='w', dir=target_dir, delete=False, suffix='.tmp') as tf:
    tf.write(new_content)
    tmp = tf.name
os.replace(tmp, target)
PYEOF

    info "Installed Orkestra skill in ~/.claude/CLAUDE.md"
  fi
  registered+=("Claude Code")
else
  skipped+=("Claude Code (claude not found)")
fi

## Cursor ##
# MCP server: registered in project-level .cursor/mcp.json
# Skill: installed globally in ~/.cursor/rules/ (Cursor 0.43+ global rules)
CURSOR_MCP="$TARGET_DIR/.cursor/mcp.json"
if [[ -d "$TARGET_DIR/.cursor" ]] || command -v cursor >/dev/null 2>&1; then
  mkdir -p "$TARGET_DIR/.cursor"

  # Write/merge project-level .cursor/mcp.json
  if [[ -f "$CURSOR_MCP" ]]; then
    if ! grep -q "\"$MCP_NAME\"" "$CURSOR_MCP" 2>/dev/null; then
      python3 - <<PYEOF 2>/dev/null || warn "Could not merge .cursor/mcp.json — add manually"
import json
with open('$CURSOR_MCP') as f: d = json.load(f)
d.setdefault('mcpServers', {})['$MCP_NAME'] = {'url': '$SSE_URL'}
with open('$CURSOR_MCP', 'w') as f: json.dump(d, f, indent=2)
PYEOF
    fi
  else
    cat > "$CURSOR_MCP" <<EOF
{
  "mcpServers": {
    "${MCP_NAME}": {
      "url": "${SSE_URL}"
    }
  }
}
EOF
  fi

  # Install skill globally: ~/.cursor/rules/orkestra.mdc (applies to all Cursor projects)
  CURSOR_GLOBAL_RULES="${HOME}/.cursor/rules"
  if [[ -d "$CURSOR_GLOBAL_RULES" ]] || mkdir -p "$CURSOR_GLOBAL_RULES" 2>/dev/null; then
    [[ -f "$SKILL_SRC" ]] && cp "$SKILL_SRC" "$CURSOR_GLOBAL_RULES/orkestra.mdc"
    info "Installed Orkestra skill in ~/.cursor/rules/orkestra.mdc"
  fi
  registered+=("Cursor")
else
  skipped+=("Cursor (.cursor/ dir not found)")
fi

## GitHub Copilot (VS Code) ##
VSCODE_MCP="$TARGET_DIR/.vscode/mcp.json"
if [[ -d "$TARGET_DIR/.vscode" ]] || [[ -d "$TARGET_DIR/.github" ]]; then
  mkdir -p "$TARGET_DIR/.vscode"
  if [[ ! -f "$VSCODE_MCP" ]]; then
    cat > "$VSCODE_MCP" <<EOF
{
  "servers": {
    "${MCP_NAME}": {
      "type": "http",
      "url": "${SSE_URL}"
    }
  }
}
EOF
  elif ! grep -q "\"$MCP_NAME\"" "$VSCODE_MCP" 2>/dev/null; then
    python3 - <<PYEOF 2>/dev/null || warn "Could not merge .vscode/mcp.json — add manually"
import json
with open('$VSCODE_MCP') as f: d = json.load(f)
d.setdefault('servers', {})['$MCP_NAME'] = {'type': 'http', 'url': '$SSE_URL'}
with open('$VSCODE_MCP', 'w') as f: json.dump(d, f, indent=2)
PYEOF
  fi

  COPILOT_INST="$TARGET_DIR/.github/copilot-instructions.md"
  mkdir -p "$TARGET_DIR/.github"
  if [[ -f "$SKILL_SRC" ]]; then
    # Inject skill into project-level copilot instructions (no standard global path for Copilot)
    if ! grep -q "orkestra-skill-begin" "$COPILOT_INST" 2>/dev/null; then
      [[ -f "$COPILOT_INST" ]] && cp "$COPILOT_INST" "${COPILOT_INST}.bak"
      python3 - "$COPILOT_INST" "$SKILL_SRC" <<'PYEOF' 2>/dev/null || { printf '\n\n'; cat "$SKILL_SRC"; } >> "$COPILOT_INST"
import sys, os, tempfile
target, skill_src = sys.argv[1], sys.argv[2]
with open(skill_src) as f: skill_content = f.read().strip()
begin_marker, end_marker = "<!-- orkestra-skill-begin -->", "<!-- orkestra-skill-end -->"
skill_block = f"{begin_marker}\n{skill_content}\n{end_marker}"
if os.path.exists(target):
    with open(target) as f: existing = f.read()
    new_content = existing.rstrip('\n') + '\n\n' + skill_block + '\n'
else:
    new_content = skill_block + '\n'
target_dir = os.path.dirname(os.path.abspath(target))
os.makedirs(target_dir, exist_ok=True)
with tempfile.NamedTemporaryFile(mode='w', dir=target_dir, delete=False, suffix='.tmp') as tf:
    tf.write(new_content); tmp = tf.name
os.replace(tmp, target)
PYEOF
    fi
  fi
  registered+=("GitHub Copilot")
else
  skipped+=("GitHub Copilot (.vscode/ or .github/ not found)")
fi

## Windsurf ##
WINDSURF_CFG="${HOME}/.codeium/windsurf/mcp_config.json"
if [[ -f "$WINDSURF_CFG" ]] || [[ -d "${HOME}/.codeium" ]]; then
  mkdir -p "${HOME}/.codeium/windsurf"
  if [[ -f "$WINDSURF_CFG" ]]; then
    if ! grep -q "\"$MCP_NAME\"" "$WINDSURF_CFG" 2>/dev/null; then
      python3 - <<PYEOF 2>/dev/null || warn "Could not merge Windsurf config — add manually"
import json
with open('$WINDSURF_CFG') as f: d = json.load(f)
d.setdefault('mcpServers', {})['$MCP_NAME'] = {'serverUrl': '$SSE_URL'}
with open('$WINDSURF_CFG', 'w') as f: json.dump(d, f, indent=2)
PYEOF
    fi
  else
    cat > "$WINDSURF_CFG" <<EOF
{
  "mcpServers": {
    "${MCP_NAME}": {
      "serverUrl": "${SSE_URL}"
    }
  }
}
EOF
  fi

  # Install skill globally: ~/.codeium/windsurf/global_rules.md (applies to all Windsurf projects)
  WINDSURF_GLOBAL_RULES="${HOME}/.codeium/windsurf/global_rules.md"
  if [[ -f "$SKILL_SRC" ]]; then
    [[ -f "$WINDSURF_GLOBAL_RULES" ]] && cp "$WINDSURF_GLOBAL_RULES" "${WINDSURF_GLOBAL_RULES}.bak"
    python3 - "$WINDSURF_GLOBAL_RULES" "$SKILL_SRC" <<'PYEOF' 2>/dev/null || warn "Could not update Windsurf global_rules.md — add manually"
import sys, os, tempfile
target, skill_src = sys.argv[1], sys.argv[2]
with open(skill_src) as f: skill_content = f.read().strip()
begin_marker, end_marker = "<!-- orkestra-skill-begin -->", "<!-- orkestra-skill-end -->"
skill_block = f"{begin_marker}\n{skill_content}\n{end_marker}"
if os.path.exists(target):
    with open(target) as f: existing = f.read()
    if begin_marker in existing and end_marker in existing:
        before = existing[:existing.index(begin_marker)]
        after  = existing[existing.index(end_marker) + len(end_marker):]
        new_content = before.rstrip('\n') + '\n\n' + skill_block + '\n' + after.lstrip('\n')
    else:
        new_content = existing.rstrip('\n') + '\n\n' + skill_block + '\n'
else:
    new_content = skill_block + '\n'
target_dir = os.path.dirname(os.path.abspath(target))
os.makedirs(target_dir, exist_ok=True)
with tempfile.NamedTemporaryFile(mode='w', dir=target_dir, delete=False, suffix='.tmp') as tf:
    tf.write(new_content); tmp = tf.name
os.replace(tmp, target)
PYEOF
    info "Installed Orkestra skill in ~/.codeium/windsurf/global_rules.md"
  fi
  registered+=("Windsurf")
else
  skipped+=("Windsurf (~/.codeium not found)")
fi

## Continue.dev ##
CONTINUE_CFG="${HOME}/.continue/config.json"
if [[ -f "$CONTINUE_CFG" ]]; then
  if ! grep -q "\"$MCP_NAME\"" "$CONTINUE_CFG" 2>/dev/null; then
    python3 - <<PYEOF 2>/dev/null || warn "Could not merge Continue.dev config — add manually"
import json
with open('$CONTINUE_CFG') as f: d = json.load(f)
d.setdefault('experimental', {}).setdefault('modelContextProtocolServers', [])
existing = [s for s in d['experimental']['modelContextProtocolServers'] if s.get('name') == '$MCP_NAME']
if not existing:
    d['experimental']['modelContextProtocolServers'].append({'url': '$SSE_URL', 'name': '$MCP_NAME'})
with open('$CONTINUE_CFG', 'w') as f: json.dump(d, f, indent=2)
PYEOF
  fi
  registered+=("Continue.dev")
else
  skipped+=("Continue.dev (~/.continue/config.json not found)")
fi

## Zed ##
ZED_SETTINGS="${HOME}/.config/zed/settings.json"
if [[ -f "$ZED_SETTINGS" ]]; then
  if ! grep -q "\"$MCP_NAME\"" "$ZED_SETTINGS" 2>/dev/null; then
    python3 - <<PYEOF 2>/dev/null || warn "Could not merge Zed settings — add manually"
import json
with open('$ZED_SETTINGS') as f: d = json.load(f)
d.setdefault('context_servers', {})['$MCP_NAME'] = {'url': '$SSE_URL'}
with open('$ZED_SETTINGS', 'w') as f: json.dump(d, f, indent=2)
PYEOF
  fi
  registered+=("Zed")
else
  skipped+=("Zed (~/.config/zed/settings.json not found)")
fi

# ---------- summary ----------

echo ""
info "=== Orkestra Setup Complete ==="
echo ""
printf "  Project ID : %s\n" "$PROJECT_ID"
printf "  URL        : %s\n" "$BASE_URL"
[[ -n "$MCP_TOKEN" ]] && printf "  MCP Token  : %s\n" "$MCP_TOKEN"
echo ""

if [[ ${#registered[@]} -gt 0 ]]; then
  info "Registered with:"
  for tool in "${registered[@]}"; do
    printf "  ✓ %s\n" "$tool"
  done
fi
echo ""

if [[ ${#skipped[@]} -gt 0 ]]; then
  warn "Skipped (not detected):"
  for tool in "${skipped[@]}"; do
    printf "  ✗ %s\n" "$tool"
  done
fi

echo ""
info "Run 'docker compose logs -f' to tail server logs."
info "Health: curl ${BASE_URL}/health"
