#!/usr/bin/env bash
set -euo pipefail

# Orkestra install script: repo bootstrap, Docker runtime, global skill install,
# and MCP registration for supported AI tools.
#
# Remote:
#   curl -fsSL https://raw.githubusercontent.com/Vijay431/Orkestra/main/install.sh | bash
#   curl -fsSL https://raw.githubusercontent.com/Vijay431/Orkestra/main/install.sh | PROJECT_ID=myapp bash
#
# Local:
#   PROJECT_ID=myapp ./install.sh

info()  { printf '\033[32m[orkestra]\033[0m %s\n' "$*"; }
warn()  { printf '\033[33m[orkestra]\033[0m %s\n' "$*"; }
error() { printf '\033[31m[orkestra]\033[0m %s\n' "$*" >&2; }
die()   { error "$*"; exit 1; }
step()  { info "Step $1/8: $2"; }

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "Required command not found: $1"
}

random_slug() {
  local suffix
  if command -v openssl >/dev/null 2>&1; then
    suffix="$(openssl rand -hex 4 | tr -dc 'a-f0-9' | cut -c1-6)"
  else
    suffix="$(printf '%s' "$$-$(date +%s%N)" | cksum | awk '{print $1}' | cut -c1-6)"
  fi
  printf 'orkestra-%s' "$suffix"
}

is_interactive() {
  [[ -t 0 && -t 1 ]]
}

compose_logs_cmd() {
  printf 'docker compose -f %q --env-file %q logs -f' "$COMPOSE_FILE" "$ENV_FILE"
}

die_health() {
  error "Orkestra did not become healthy within 30s."
  error "Inspect logs with: $(compose_logs_cmd)"
  exit 1
}

upsert_json_mcp() {
  local file="$1" root_key="$2" name="$3" url_key="$4" url="$5" type_value="${6:-}" header_mode="${7:-none}"
  mkdir -p "$(dirname "$file")"
  python3 - "$file" "$root_key" "$name" "$url_key" "$url" "$type_value" "$header_mode" "${MCP_TOKEN:-}" <<'PYEOF'
import json
import os
import sys
import tempfile

path, root_key, name, url_key, url, type_value, header_mode, token = sys.argv[1:]

if os.path.exists(path) and os.path.getsize(path) > 0:
    with open(path) as f:
        data = json.load(f)
else:
    data = {}

server = {url_key: url}
if type_value:
    server["type"] = type_value
if token and header_mode == "headers":
    server["headers"] = {"Authorization": f"Bearer {token}"}
elif token and header_mode == "requestHeaders":
    server["requestHeaders"] = {"Authorization": f"Bearer {token}"}

if root_key == "continue":
    experimental = data.setdefault("experimental", {})
    servers = experimental.setdefault("modelContextProtocolServers", [])
    servers[:] = [s for s in servers if s.get("name") != name]
    entry = {"name": name, "url": url}
    if token:
        entry["requestHeaders"] = {"Authorization": f"Bearer {token}"}
    servers.append(entry)
else:
    data.setdefault(root_key, {})[name] = server

directory = os.path.dirname(os.path.abspath(path))
os.makedirs(directory, exist_ok=True)
with tempfile.NamedTemporaryFile(mode="w", dir=directory, delete=False, suffix=".tmp") as tmp:
    json.dump(data, tmp, indent=2)
    tmp.write("\n")
    tmp_path = tmp.name
os.replace(tmp_path, path)
PYEOF
}

sync_dir() {
  local src="$1" dst="$2"
  mkdir -p "$dst"
  if command -v rsync >/dev/null 2>&1; then
    rsync -a --delete "$src/" "$dst/"
  else
    rm -rf "$dst"
    mkdir -p "$dst"
    cp -R "$src/." "$dst/"
  fi
}

install_skill_file_block() {
  local target="$1" source="$2" missing_message="$3"
  [[ -f "$source" ]] || return 0
  mkdir -p "$(dirname "$target")"
  [[ -f "$target" ]] && cp "$target" "${target}.bak"
  python3 - "$target" "$source" <<'PYEOF' 2>/dev/null || warn "$missing_message"
import os
import sys
import tempfile

target, source = sys.argv[1], sys.argv[2]
with open(source) as f:
    skill = f.read().strip()

begin = "<!-- orkestra-skill-begin -->"
end = "<!-- orkestra-skill-end -->"
block = f"{begin}\n{skill}\n{end}"

if os.path.exists(target):
    with open(target) as f:
        existing = f.read()
    if begin in existing and end in existing:
        before = existing[:existing.index(begin)]
        after = existing[existing.index(end) + len(end):]
        content = before.rstrip("\n") + "\n\n" + block + "\n" + after.lstrip("\n")
    else:
        content = existing.rstrip("\n") + "\n\n" + block + "\n"
else:
    content = block + "\n"

directory = os.path.dirname(os.path.abspath(target))
os.makedirs(directory, exist_ok=True)
with tempfile.NamedTemporaryFile(mode="w", dir=directory, delete=False, suffix=".tmp") as tmp:
    tmp.write(content)
    tmp_path = tmp.name
os.replace(tmp_path, target)
PYEOF
}

link_skill_dir() {
  local target_dir="$1"
  [[ -d "$target_dir" ]] || return 0
  local link="$target_dir/orkestra"
  local rel_target
  rel_target="$(python3 - "$target_dir" "$SKILL_INSTALL_DIR" <<'PYEOF' 2>/dev/null || true
import os
import sys
print(os.path.relpath(sys.argv[2], sys.argv[1]))
PYEOF
)"
  [[ -n "$rel_target" ]] || rel_target="$SKILL_INSTALL_DIR"
  if [[ -L "$link" ]]; then
    ln -sf "$rel_target" "$link"
  elif [[ ! -e "$link" ]]; then
    ln -s "$rel_target" "$link"
  fi
}

detect_tool() {
  case "$1" in
    claude) command -v claude >/dev/null 2>&1 ;;
    codex) command -v codex >/dev/null 2>&1 || [[ -d "${CODEX_HOME:-$HOME/.codex}" ]] ;;
    cursor) command -v cursor >/dev/null 2>&1 || [[ -d "$TARGET_DIR/.cursor" ]] || [[ -d "$HOME/.cursor" ]] ;;
    copilot) [[ -d "$TARGET_DIR/.vscode" ]] || [[ -d "$TARGET_DIR/.github" ]] || command -v code >/dev/null 2>&1 ;;
    windsurf) command -v windsurf >/dev/null 2>&1 || [[ -d "$HOME/.codeium" ]] ;;
    zed) command -v zed >/dev/null 2>&1 || [[ -d "$HOME/.config/zed" ]] ;;
    continue) command -v continue >/dev/null 2>&1 || [[ -d "$HOME/.continue" ]] ;;
    *) return 1 ;;
  esac
}

display_tool() {
  case "$1" in
    claude) printf 'Claude Code' ;;
    codex) printf 'Codex' ;;
    cursor) printf 'Cursor' ;;
    copilot) printf 'GitHub Copilot' ;;
    windsurf) printf 'Windsurf' ;;
    zed) printf 'Zed' ;;
    continue) printf 'Continue.dev' ;;
    *) printf '%s' "$1" ;;
  esac
}

select_tools() {
  local supported=(claude codex cursor copilot windsurf zed continue)
  selected_tools=()
  detected_tools=()
  skipped_tools=()

  for tool in "${supported[@]}"; do
    if detect_tool "$tool"; then
      detected_tools+=("$tool")
    fi
  done

  if [[ -n "${ORKESTRA_TOOLS:-}" ]]; then
    local requested="${ORKESTRA_TOOLS,,}"
    requested="${requested// /}"
    if [[ "$requested" == "all" ]]; then
      selected_tools=("${supported[@]}")
    else
      IFS=',' read -r -a selected_tools <<< "$requested"
    fi
    return
  fi

  if is_interactive && [[ ${#detected_tools[@]} -gt 0 ]]; then
    echo ""
    info "Detected AI tools:"
    local i=1
    for tool in "${detected_tools[@]}"; do
      printf "  %d) %s\n" "$i" "$(display_tool "$tool")"
      i=$((i + 1))
    done
    echo ""
    read -rp "Select tools to configure [all, comma-separated numbers/names, none] (all): " tool_input
    tool_input="${tool_input:-all}"
    tool_input="${tool_input,,}"
    tool_input="${tool_input// /}"
    if [[ "$tool_input" == "none" ]]; then
      selected_tools=()
    elif [[ "$tool_input" == "all" ]]; then
      selected_tools=("${detected_tools[@]}")
    else
      IFS=',' read -r -a raw_choices <<< "$tool_input"
      for choice in "${raw_choices[@]}"; do
        if [[ "$choice" =~ ^[0-9]+$ ]] && (( choice >= 1 && choice <= ${#detected_tools[@]} )); then
          selected_tools+=("${detected_tools[$((choice - 1))]}")
        else
          selected_tools+=("$choice")
        fi
      done
    fi
  else
    selected_tools=("${detected_tools[@]}")
  fi
}

register_claude() {
  if ! command -v claude >/dev/null 2>&1; then
    skipped+=("Claude Code (claude command not found)")
    return
  fi
  local args=(mcp add "$MCP_NAME" --transport http "$SSE_URL")
  if [[ -n "${MCP_TOKEN:-}" ]]; then
    args+=(--header "Authorization: Bearer ${MCP_TOKEN}")
  fi
  claude "${args[@]}" 2>/dev/null || warn "Could not register Claude Code MCP automatically; add ${SSE_URL} manually."
  install_skill_file_block "$HOME/.claude/CLAUDE.md" "$SKILL_SRC" "Could not update ~/.claude/CLAUDE.md; add skill/SKILL.md manually."
  registered+=("Claude Code")
}

register_codex() {
  local codex_home="${CODEX_HOME:-$HOME/.codex}"
  sync_dir "$SKILL_DIR" "$codex_home/skills/orkestra"
  mkdir -p "$codex_home"
  local config="$codex_home/config.toml"
  python3 - "$config" "$MCP_NAME" "$SSE_URL" "${MCP_TOKEN:-}" <<'PYEOF' 2>/dev/null || warn "Could not update Codex config; add [mcp_servers] entry manually."
import os
import re
import sys
import tempfile

path, name, url, token = sys.argv[1:]
content = ""
if os.path.exists(path):
    with open(path) as f:
        content = f.read()

block_lines = [
    f"[mcp_servers.{name}]",
    f'url = "{url}"',
]
if token:
    block_lines.append('[mcp_servers.%s.headers]' % name)
    block_lines.append(f'Authorization = "Bearer {token}"')
block = "\n".join(block_lines)

pattern = re.compile(rf"(?ms)^\[mcp_servers\.{re.escape(name)}\]\n.*?(?=^\[(?!mcp_servers\.{re.escape(name)}(?:\.|\]))|\Z)")
if pattern.search(content):
    content = pattern.sub(block + "\n", content)
else:
    if "[mcp_servers]" not in content:
        content = content.rstrip() + "\n\n[mcp_servers]\n"
    content = content.rstrip() + "\n\n" + block + "\n"

directory = os.path.dirname(os.path.abspath(path))
os.makedirs(directory, exist_ok=True)
with tempfile.NamedTemporaryFile(mode="w", dir=directory, delete=False, suffix=".tmp") as tmp:
    tmp.write(content.lstrip("\n"))
    tmp_path = tmp.name
os.replace(tmp_path, path)
PYEOF
  registered+=("Codex")
}

register_cursor() {
  local config="$TARGET_DIR/.cursor/mcp.json"
  upsert_json_mcp "$config" "mcpServers" "$MCP_NAME" "url" "$SSE_URL" "" "headers" || {
    skipped+=("Cursor (could not write $config)")
    return
  }
  mkdir -p "$HOME/.cursor/rules"
  [[ -f "$SKILL_SRC" ]] && cp "$SKILL_SRC" "$HOME/.cursor/rules/orkestra.mdc"
  registered+=("Cursor")
}

register_copilot() {
  local config="$TARGET_DIR/.vscode/mcp.json"
  upsert_json_mcp "$config" "servers" "$MCP_NAME" "url" "$SSE_URL" "http" "headers" || {
    skipped+=("GitHub Copilot (could not write $config)")
    return
  }
  install_skill_file_block "$TARGET_DIR/.github/copilot-instructions.md" "$SKILL_SRC" "Could not update .github/copilot-instructions.md; add skill/SKILL.md manually."
  registered+=("GitHub Copilot")
}

register_windsurf() {
  local config="$HOME/.codeium/windsurf/mcp_config.json"
  upsert_json_mcp "$config" "mcpServers" "$MCP_NAME" "serverUrl" "$SSE_URL" "" "headers" || {
    skipped+=("Windsurf (could not write $config)")
    return
  }
  install_skill_file_block "$HOME/.codeium/windsurf/global_rules.md" "$SKILL_SRC" "Could not update Windsurf global_rules.md; add skill/SKILL.md manually."
  registered+=("Windsurf")
}

register_zed() {
  local config="$HOME/.config/zed/settings.json"
  if [[ -n "${MCP_TOKEN:-}" ]]; then
    warn "Zed config header support varies; if auth fails, add Authorization: Bearer ${MCP_TOKEN} manually for ${MCP_NAME}."
  fi
  upsert_json_mcp "$config" "context_servers" "$MCP_NAME" "url" "$SSE_URL" "" "none" || {
    skipped+=("Zed (could not write $config)")
    return
  }
  registered+=("Zed")
}

register_continue() {
  local config="$HOME/.continue/config.json"
  upsert_json_mcp "$config" "continue" "$MCP_NAME" "url" "$SSE_URL" "" "requestHeaders" || {
    skipped+=("Continue.dev (could not write $config)")
    return
  }
  registered+=("Continue.dev")
}

register_tool() {
  local tool="$1"
  case "$tool" in
    claude) register_claude ;;
    codex) register_codex ;;
    cursor) register_cursor ;;
    copilot) register_copilot ;;
    windsurf) register_windsurf ;;
    zed) register_zed ;;
    continue) register_continue ;;
    "") ;;
    *) skipped+=("$(display_tool "$tool") (unsupported tool name)") ;;
  esac
}

ORKESTRA_REPO="${ORKESTRA_REPO:-https://github.com/Vijay431/Orkestra}"
ORKESTRA_HOME="${ORKESTRA_HOME:-$HOME/.orkestra}"

step 1 "checking prerequisites"
require_cmd docker
require_cmd curl
require_cmd python3
docker compose version >/dev/null 2>&1 || die "docker compose plugin is required"

step 2 "preparing repository"
raw_source="${BASH_SOURCE[0]:-}"
if [[ -n "$raw_source" && "$raw_source" != "bash" && "$raw_source" != "/dev/stdin" ]]; then
  SCRIPT_DIR="$(cd "$(dirname "$raw_source")" && pwd)"
else
  SCRIPT_DIR=""
fi

if [[ -z "$SCRIPT_DIR" || ! -f "$SCRIPT_DIR/Dockerfile" || ! -d "$SCRIPT_DIR/skill" ]]; then
  require_cmd git
  if [[ -d "$ORKESTRA_HOME/.git" ]]; then
    info "Updating existing Orkestra clone at $ORKESTRA_HOME"
    git -C "$ORKESTRA_HOME" pull --ff-only --quiet
  else
    info "Cloning Orkestra into $ORKESTRA_HOME"
    git clone --depth 1 --quiet "$ORKESTRA_REPO" "$ORKESTRA_HOME"
  fi
  SCRIPT_DIR="$ORKESTRA_HOME"
fi
info "Using repo at $SCRIPT_DIR"

step 3 "writing environment"
PROJECT_ID="${PROJECT_ID:-}"
if [[ -z "$PROJECT_ID" ]]; then
  PROJECT_ID="$(random_slug)"
  info "Generated PROJECT_ID=$PROJECT_ID"
else
  info "Using PROJECT_ID=$PROJECT_ID"
fi

PORT="${PORT:-8080}"
WEB_PORT="${WEB_PORT:-7777}"
TARGET_DIR="${TARGET_DIR:-$(pwd)}"
mkdir -p "$TARGET_DIR"

MCP_TOKEN="${MCP_TOKEN:-}"
if [[ -z "$MCP_TOKEN" ]] && command -v openssl >/dev/null 2>&1; then
  MCP_TOKEN="$(openssl rand -hex 32)"
  info "Generated MCP_TOKEN (stored in .env)"
fi

COMPOSE_FILE="$TARGET_DIR/docker-compose.yml"
if [[ ! -f "$COMPOSE_FILE" ]]; then
  cp "$SCRIPT_DIR/docker-compose.yml" "$COMPOSE_FILE"
  info "Copied docker-compose.yml to $COMPOSE_FILE"
fi

ENV_FILE="$TARGET_DIR/.env"
{
  printf 'PROJECT_ID=%s\n' "$PROJECT_ID"
  printf 'PORT=%s\n' "$PORT"
  printf 'WEB_PORT=%s\n' "$WEB_PORT"
  printf 'MCP_TOKEN=%s\n' "$MCP_TOKEN"
  printf 'BIND_ADDR=0.0.0.0\n'
  printf 'ORKESTRA_BUILD_CONTEXT=%s\n' "$SCRIPT_DIR"
} > "$ENV_FILE"
info "Wrote $ENV_FILE"

COMPOSE_FLAGS=(-f "$COMPOSE_FILE" --env-file "$ENV_FILE")

step 4 "building Docker image"
docker compose "${COMPOSE_FLAGS[@]}" build

step 5 "starting container"
docker compose "${COMPOSE_FLAGS[@]}" up -d

step 6 "waiting for health"
BASE_URL="http://localhost:${PORT}"
DEADLINE=$(($(date +%s) + 30))
until curl -sf "${BASE_URL}/health" >/dev/null 2>&1; do
  [[ $(date +%s) -lt $DEADLINE ]] || die_health
  sleep 1
done
info "Health check passed at ${BASE_URL}/health"

step 7 "installing global skill"
SKILL_DIR="$SCRIPT_DIR/skill"
SKILL_SRC="$SKILL_DIR/SKILL.md"
GLOBAL_SKILLS_HUB="${HOME}/.agents/skills"
SKILL_INSTALL_DIR="${GLOBAL_SKILLS_HUB}/orkestra"

if [[ -d "$SKILL_DIR" ]]; then
  sync_dir "$SKILL_DIR" "$SKILL_INSTALL_DIR"
  link_skill_dir "$HOME/.claude/skills"
  link_skill_dir "$HOME/.cursor/skills"
  link_skill_dir "$HOME/.codeium/windsurf/skills"
  link_skill_dir "$HOME/.continue/skills"
  link_skill_dir "${CODEX_HOME:-$HOME/.codex}/skills"
  info "Installed skill to $SKILL_INSTALL_DIR"
else
  warn "skill/ directory not found; skipping skill installation"
fi

step 8 "registering MCP with selected AI tools"
SSE_URL="${BASE_URL}/sse"
MCP_NAME="orkestra-${PROJECT_ID}"
registered=()
skipped=()
selected_tools=()
detected_tools=()
select_tools

if [[ ${#selected_tools[@]} -eq 0 ]]; then
  warn "No AI tools selected for MCP registration."
else
  for tool in "${selected_tools[@]}"; do
    register_tool "$tool"
  done
fi

if [[ -z "${ORKESTRA_TOOLS:-}" ]] && ! is_interactive; then
  for tool in claude codex cursor copilot windsurf zed continue; do
    detect_tool "$tool" || skipped+=("$(display_tool "$tool") (not detected)")
  done
fi

echo ""
info "=== Orkestra Setup Complete ==="
echo ""
printf "  Project ID : %s\n" "$PROJECT_ID"
printf "  Kanban UI  : http://localhost:%s\n" "$WEB_PORT"
printf "  Health     : %s/health\n" "$BASE_URL"
printf "  MCP SSE    : %s\n" "$SSE_URL"
[[ -n "$MCP_TOKEN" ]] && printf "  MCP Token  : %s\n" "$MCP_TOKEN"
echo ""

if [[ ${#registered[@]} -gt 0 ]]; then
  info "Registered with:"
  for tool in "${registered[@]}"; do
    printf "  - %s\n" "$tool"
  done
fi

if [[ ${#skipped[@]} -gt 0 ]]; then
  echo ""
  warn "Skipped:"
  for tool in "${skipped[@]}"; do
    printf "  - %s\n" "$tool"
  done
fi

echo ""
info "Tail logs with: $(compose_logs_cmd)"
