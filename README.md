# google-tag-manager-mcp

A [Model Context Protocol (MCP)](https://modelcontextprotocol.io) server for the **Google Tag Manager API**, built for personal use with Claude Desktop and Claude Code.

Runs as a local **stdio subprocess** — no hosted server, no OAuth proxy, no Docker. Auth is handled once via a browser flow and tokens are stored locally.

> **Built by [Max Västhav](https://github.com/VasthavM)**
> Derived from [paolobietolini/gtm-mcp-server](https://github.com/paolobietolini/gtm-mcp-server) (MIT) — the original HTTP server with OAuth proxy was rewritten as a stdio-based personal-use server.

---

## What you can do

Once connected, Claude can read and manage your entire GTM workspace through natural language.

### 50 Tools

**Reading data**
| Tool | Description |
|------|-------------|
| `list_accounts` | List all GTM accounts |
| `list_containers` | List containers in an account |
| `list_workspaces` | List workspaces in a container |
| `get_workspace_status` | Get pending changes in a workspace |
| `list_tags` / `get_tag` | List or inspect tags |
| `list_triggers` / `get_trigger` | List or inspect triggers |
| `list_variables` / `get_variable` | List or inspect variables |
| `list_folders` / `get_folder_entities` | Browse folder contents |
| `list_templates` / `get_template` | List or inspect custom templates |
| `list_versions` | List container versions |
| `list_built_in_variables` | List enabled built-in variables |
| `list_clients` / `get_client` | List or inspect server-side clients |
| `list_transformations` / `get_transformation` | List or inspect transformations |
| `get_tag_templates` / `get_trigger_templates` | Get correct parameter formats for tags/triggers |

**Creating & updating**
| Tool | Description |
|------|-------------|
| `create_tag` / `update_tag` | Create or update a tag |
| `create_trigger` / `update_trigger` | Create or update a trigger |
| `create_variable` / `update_variable` | Create or update a variable |
| `create_workspace` | Create a new workspace |
| `create_container` | Create a new container |
| `create_template` / `update_template` | Create or update a custom template |
| `create_client` / `update_client` | Create or update a server-side client |
| `create_transformation` / `update_transformation` | Create or update a transformation |
| `enable_built_in_variables` | Enable built-in variables |
| `import_gallery_template` | Import a template from the Community Gallery |

**Publishing**
| Tool | Description |
|------|-------------|
| `create_version` | Create a container version |
| `publish_version` | Publish a version to live |

**Deleting** *(all require `confirm: true`)*
| Tool | Description |
|------|-------------|
| `delete_tag` | Delete a tag |
| `delete_trigger` | Delete a trigger |
| `delete_variable` | Delete a variable |
| `delete_template` | Delete a custom template |
| `delete_container` | Delete a container |
| `delete_client` | Delete a server-side client |
| `delete_transformation` | Delete a transformation |
| `disable_built_in_variables` | Disable built-in variables |

**Utility**
| Tool | Description |
|------|-------------|
| `ping` | Test server connectivity |
| `auth_status` | Confirm authentication is active |

### 6 Resources

URI-based read access to live GTM data:

- `gtm://accounts` — All accounts
- `gtm://accounts/{accountId}/containers` — Containers in an account
- `gtm://accounts/{accountId}/containers/{containerId}/workspaces` — Workspaces
- `gtm://accounts/{accountId}/containers/{containerId}/workspaces/{workspaceId}/tags` — Tags
- `gtm://accounts/{accountId}/containers/{containerId}/workspaces/{workspaceId}/triggers` — Triggers
- `gtm://accounts/{accountId}/containers/{containerId}/workspaces/{workspaceId}/variables` — Variables

### 4 Prompts

Guided multi-step workflows Claude can follow:

| Prompt | Description |
|--------|-------------|
| `audit_container` | Full audit of a container's tags, triggers, and variables |
| `generate_tracking_plan` | Generate a tracking plan from business goals |
| `suggest_ga4_setup` | Suggest a GA4 tag/trigger/variable setup |
| `find_gallery_template` | Find and import a Community Gallery template |

---

## Setup

### Prerequisites

- Go 1.24+ — install from [go.dev/dl](https://go.dev/dl)
- A Google Cloud project with the **Tag Manager API** enabled
- OAuth 2.0 credentials (Desktop app type)

### 1. Clone and build

```bash
git clone https://github.com/VasthavM/google-tag-manager-mcp
cd google-tag-manager-mcp
go build -o google-tag-manager-mcp .
```

### 2. Create Google Cloud credentials

1. Go to [Google Cloud Console](https://console.cloud.google.com/) → **APIs & Services** → **Enable APIs and Services**
2. Search for **Tag Manager API** and enable it
3. Go to **APIs & Services** → **Credentials** → **Create Credentials** → **OAuth client ID**
4. Choose **Desktop app** as the application type
5. Download the JSON file

### 3. Save your credentials

```bash
mkdir -p ~/.config/google-tag-manager-mcp
cp ~/Downloads/client_secret_*.json ~/.config/google-tag-manager-mcp/credentials.json
```

### 4. Authenticate (one-time browser flow)

```bash
./google-tag-manager-mcp
```

The server will print a URL. Open it in your browser, grant GTM access, and the token is saved automatically to `~/.config/google-tag-manager-mcp/token.json`. After that, the server starts up — you can close it with `Ctrl+C`.

Tokens refresh automatically. You won't need to repeat this unless you revoke access.

### 5. Register with Claude Code

```bash
claude mcp add --scope user google-tag-manager /path/to/google-tag-manager-mcp
```

Or for **Claude Desktop**, add to `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS) or `%APPDATA%\Claude\claude_desktop_config.json` (Windows):

```json
{
  "mcpServers": {
    "google-tag-manager": {
      "command": "/path/to/google-tag-manager-mcp"
    }
  }
}
```

---

## Configuration

| Environment variable | Default | Description |
|----------------------|---------|-------------|
| `GTM_MCP_CREDENTIALS` | `~/.config/google-tag-manager-mcp/credentials.json` | Path to credentials file |
| `GTM_MCP_TOKEN` | `~/.config/google-tag-manager-mcp/token.json` | Path to saved token |
| `GTM_MCP_CONFIG_DIR` | `~/.config/google-tag-manager-mcp/` | Override config directory |
| `GTM_DEBUG` | _(unset)_ | Set to any value to log HTTP request/response bodies (localhost only) |

---

## Security

- `credentials.json` and `token.json` are listed in `.gitignore` and must never be committed
- The token file is stored with `0600` permissions (owner read/write only)
- All destructive operations (`delete_*`, `disable_built_in_variables`, `publish_version`) require an explicit `confirm: true` parameter
- All update operations use fingerprint-based concurrency control to prevent accidental overwrites

---

## Attribution

The GTM API client, tool handlers, resources, and prompts are derived from [paolobietolini/gtm-mcp-server](https://github.com/paolobietolini/gtm-mcp-server) (MIT License).

This fork rewrites the transport layer: the original HTTP server with OAuth 2.1 proxy (designed for hosted multi-user deployment) is replaced with a stdio transport and local OAuth2 browser flow, making it suitable for personal use with Claude Desktop and Claude Code.

---

## License

MIT — see [LICENSE](LICENSE).
