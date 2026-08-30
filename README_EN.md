# Alchemy Furnace · 炼丹炉

> A desktop language-pattern workshop: distill reusable ways of speaking into Elixir Pills, then let Dao Agents carry those patterns into one-to-one conversations and roundtable discussions.

[中文 README](README.md) · [Repository](https://github.com/yusanwen-code/alchemy-furnace)

## Product scope

Alchemy Furnace is a **Wails desktop application**. The Go gateway, Python language engine, and Next.js static UI run locally, with SQLite as the default desktop database. Model calls use an OpenAI-compatible endpoint configured in Settings.

The desktop package is the supported product form. The Web UI, `serve` command, and Docker Compose remain available for development, diagnostics, and CI; they are not a separately supported Web or mobile-H5 product.

## Current capabilities

- **Elixir Pavilion (金丹阁)**: Manage language-pattern skill packages. The page has two tabs: “All Pills” and “Fusion Pills”. Create, edit, preview, and fuse pills, or use Nuwa Distillation to turn source material into a draft for confirmation.
- **Daoist Residence (道人府)**: Manage Dao Agent lists, names, profiles, avatars, base personalities, models, and pill bindings. Pill weight and order participate in synthesis.
- **Discourse (论道)**: Chat through the “Dialogues” and “Roundtables” tabs. One-to-one sessions are grouped under each Dao Agent; roundtables can have a topic name for fast retrieval.
- **Discourse history (论道旧录)**: Recent sessions and the full directory use authoritative Dao Agent names and do not expose Daoist UUIDs.
- **Settings**: Configure your profile and avatar, model providers, default model, and desktop options.
- **Language-pattern alchemy**: Go handles orchestration and caching; the Python engine performs structured synthesis and OpenAI-compatible model calls. A valid cache entry is reused when its source fingerprint is unchanged.

## Core concepts

| Concept | Description |
| --- | --- |
| Elixir Pill (金丹) | A structured skill package (`skill_schema`) describing a language pattern, capability, or expression tendency. |
| Dao Agent (道人) | An AI conversation persona with a base personality, name, model, and pill bindings. |
| Binding (服丹) | Attaches a pill to an agent with `weight` (0–10) and `sort_order`. |
| Synthesis (合成) | Combines base personality and bound pills into a system prompt and behavioral rules for conversation. |
| Fusion Pill (融合金丹) | Distills multiple pills into a new pill while retaining source and version metadata. |

## Architecture

```text
Wails desktop shell (static Next.js UI)
                    │ HTTP / WebSocket
                    ▼
             Go API gateway
               │          │
               │          └── SQLite (single local file by default)
               ▼
          Python language engine
       synthesis / distillation / LLM
```

On startup, the desktop app prepares its local data directory, migrates the database, and launches the bundled Python runtime. Development and self-hosted deployments can switch to PostgreSQL/MySQL and use Docker Compose to orchestrate services.

## Install and first run

Download the matching package from [GitHub Releases](https://github.com/yusanwen-code/alchemy-furnace/releases):

- macOS Apple Silicon (`darwin-arm64`)
- macOS Intel (`darwin-amd64`)
- Windows x64 (`windows-amd64`)

After launching:

1. Configure a model provider, API key, base URL, and default model in Settings.
2. Create or distill a pill in the Elixir Pavilion.
3. Create a Dao Agent and bind pills in the Daoist Residence.
4. Start a dialogue or roundtable in Discourse.

API keys are encrypted in the local database. Never commit `.env`, exported configuration, or logs.

## Local development

### Requirements

- Go (see `backend/go/go.mod`)
- Python 3.11+ and `pip`
- Node.js 20+ and `pnpm`
- Docker Desktop (only for PostgreSQL or the full Compose development stack)

### Quick start

```bash
make init                 # create .env for development/self-hosting
make dev                  # start frontend, Go, Python, and check the database
```

Run services independently when needed:

```bash
make dev-front
make dev-go
make dev-python
```

### Tests, formatting, and desktop packaging

```bash
make test
make format
make desktop-package PLATFORM=darwin-arm64 VERSION=v0.1.0
# PLATFORM: darwin-arm64, darwin-amd64, or windows-amd64
```

`make build/up/down` target the Docker Compose development environment; they are not the installation path for desktop users.

## Repository layout

```text
backend/go/       Go API gateway, Wails entry point, data access, and services
backend/python/   FastAPI language engine, distillation, and LLM calls
frontend/         Next.js + React + Tailwind desktop UI
scripts/          Development, runtime build, desktop packaging, and checks
docs/             Architecture, deployment, operations, and design docs
specs/            Feature specifications and data contracts
```

## Further reading

- [System architecture](docs/architecture.md)
- [Desktop release process](docs/deployment/desktop-release.md)
- [Frontend notes](docs/frontend.md)
- [中文 README](README.md)

## License

This project uses the [custom license](LICENSE) in the repository. Personal, educational, and non-commercial use is allowed by default; commercial use requires written authorization from the project author.

© yusanwen-code · Alchemy Furnace
