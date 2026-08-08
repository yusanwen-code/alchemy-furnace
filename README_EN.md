# Alchemy Furnace &middot; 炼丹炉

<p align="center">
  <a href="README.md">中文</a> &middot; <b>English</b>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/React-18-61DAFB?logo=react" />
  <img src="https://img.shields.io/badge/Go-1.21-00ADD8?logo=go" />
  <img src="https://img.shields.io/badge/Python-3.11-3776AB?logo=python" />
  <img src="https://img.shields.io/badge/PostgreSQL-14-4169E1?logo=postgresql" />
  <img src="https://img.shields.io/badge/Docker-Compose-2496ED?logo=docker" />
  <img src="https://img.shields.io/badge/License-Custom-orange.svg" />
</p>

<p align="center">
  <i>"The Tao gives birth to One, One gives birth to Two, Two gives birth to Three, Three gives birth to all things — and all things can be refined into an Elixir Pill."</i>
</p>

<p align="center">
  <a href="#introduction">Introduction</a> &middot;
  <a href="#features">Features</a> &middot;
  <a href="#architecture">Architecture</a> &middot;
  <a href="#quick-start">Quick Start</a> &middot;
  <a href="#concepts">Concepts</a> &middot;
  <a href="#api">API</a> &middot;
  <a href="#documentation">Documentation</a>
</p>

---

## Introduction

**Alchemy Furnace** is a **Skill-Persona Alchemy** system inspired by Taoist inner alchemy. Here, language patterns and persona traits are refined into **Elixir Pills**, and AI Agents become **Dao Cultivators**. When a cultivator consumes a pill, their speech and thinking are transformed by the pill's "elixir nature," letting them discourse with you in an entirely new linguistic style.

> The ancients said: "When the pill is complete, dragon and tiger submit; when the Tao is attained, ghosts and gods are awed." Today we use AI as fire and language patterns as ingredients, refining pills of persona and raising cultivators of every temperament.

## About the Author

This project, **Alchemy Furnace (炼丹炉)**, is developed and maintained by **[yusanwen-code](https://github.com/yusanwen-code)**.

- **GitHub**: [https://github.com/yusanwen-code](https://github.com/yusanwen-code)
- **Repository**: [https://github.com/yusanwen-code/alchemy-furnace](https://github.com/yusanwen-code/alchemy-furnace)
- **Licensing**: Open source for personal use; closed for commercial use. Free for personal, learning, and research purposes; commercial use requires a separate license and must retain the project name and developer attribution.

## Features

### Core Capabilities

- **Pill Refinement** — Craft Elixir Pills from structured skill schemas: expression DNA, mental models, decision heuristics, taboos, honest limits, and example dialogues
- **Cultivator Creation** — Create personalized AI Agents (Dao Cultivators) with base personalities and chosen LLMs
- **Pill Consumption** — A cultivator may consume multiple pills, with adjustable weights and consumption order
- **Skill-Persona Synthesis** — The synthesis engine structurally merges the cultivator's base personality with consumed pills, then uses LLM-driven emergent inference to produce a unified "elixir nature" system prompt
- **Multi-Pill Fusion** — Multiple pills blend by weight, with automatic conflict detection (elixir clashes) and emergent rule extraction to avoid stylistic tearing
- **Pill Trial Preview** — Temporarily combine personalities and pills to preview synthesis results without binding to a cultivator
- **Tao Discourse** — Streaming chat where the cultivator always replies in the fused linguistic style; supports mid-stream stop and automatic reconnection
- **Model Management** — Independently configure models from multiple providers (OpenAI / DeepSeek / Qwen / Ollama, etc.), with encrypted API key storage, connection testing, and default/synthesis model settings

### Technical Highlights

- **Responsive Design** — A single React codebase supports both desktop web and mobile H5
- **Microservice Architecture** — Go handles the API gateway and business data; Python handles the language-pattern synthesis engine and LLM calls, with clear separation of concerns
- **Streaming Chat** — WebSocket real-time delivery with typewriter-style AI replies
- **Synthesis Cache** — Each cultivator's synthesized system prompt is cached on demand and automatically rebuilt when personality or pills change
- **Taoist Aesthetics** — Cinnabar red, gold-foil yellow, rice-paper white, and CSS-animated furnace effects for an immersive experience

## Architecture

```
                Purple Palace (React Frontend)
                         │
                  Southern Heaven Gate (Nginx)
                    /          \
            Azure Dragon     White Tiger
             (Go API)    (Python Language Engine)
              │              │
        Jade Archive (PG)   LLM (OpenAI-compatible)
```

| Service | Tech | Responsibility | Port |
|---------|------|----------------|------|
| **Purple Palace** | React 18 + TypeScript + Tailwind CSS | User interface | 80 (Nginx) |
| **Azure Dragon** | Go 1.21 + Gin + GORM | API gateway, business logic, cultivator/pill/session persistence | 8080 |
| **White Tiger** | Python 3.11 + FastAPI | Language-pattern synthesis engine, LLM calls | 8000 |
| **Jade Archive** | PostgreSQL 14 | Business data persistence | 5432 |
| **Southern Heaven Gate** | Nginx | Reverse proxy, static files | 80 |

## Quick Start

### Requirements

- [Docker](https://docs.docker.com/get-docker/) 20.10+
- [Docker Compose](https://docs.docker.com/compose/install/) 2.0+
- An OpenAI-compatible API key (OpenAI, DeepSeek, Qwen, etc.)

### One-Click Deployment

```bash
# 1. Clone the repository
git clone https://github.com/yusanwen-code/alchemy-furnace.git
cd alchemy-furnace

# 2. Configure the environment
cp .env.example .env
# Edit .env and fill in your OPENAI_API_KEY,
# and set MODEL_KEY_SECRET (the encryption key for model API keys —
# any long random string, e.g. `openssl rand -hex 32`);
# without it, API keys cannot be saved in Model Management

# 3. Light the furnace
make deploy

# 4. Open the furnace
# Visit http://localhost in your browser
```

### Make Commands

```bash
make help       # Show all available commands
make init       # Initialize the environment
make build      # Build images
make up         # Start services
make down       # Stop services
make logs       # View logs
make ps         # View service status
make clean      # Remove all data
make dev-front  # Frontend dev mode
make dev-go     # Go backend dev mode
make dev-python # Python language engine dev mode
```

### Local Development (without Docker)

Prerequisites: PostgreSQL 14+ (e.g. [Postgres.app](https://postgresapp.com/)), Go 1.21+, Python 3.11+, Node.js 20+.

```bash
# 1. Prepare the database (local PostgreSQL on 5432)
createuser alchemy && createdb -O alchemy alchemy_db
psql -c "ALTER USER alchemy PASSWORD 'alchemy123';"

# 2. Configure the environment (set DB_HOST to localhost
#    and PYTHON_ENGINE_BASE_URL to http://localhost:8000)
cp .env.example .env  # Edit for your local setup

# 3. Start in three terminals
make dev-python   # Python language engine → http://localhost:8000
make dev-go       # Go API gateway (auto-migrate + seed built-in pills) → http://localhost:8080
make dev-front    # React frontend → http://localhost:3000
```

### First-Time Guide

1. **Refine a Pill** — Go to the Pill Chamber and create an Elixir Pill: fill in expression DNA (sentence length, formality, vocabulary, taboo words), mental models, decision heuristics, example dialogues, and other structured content
2. **Raise a Cultivator** — Go to the Cultivator Hall, click "Accept Disciple", and configure the cultivator's name, base personality, and chosen LLM
3. **Consume Pills** — On the cultivator detail page, have the cultivator consume refined pills, adjusting weights and consumption order
4. **Trial Preview** (optional) — On the Pill Trial page, temporarily combine personalities and pills to preview synthesis results
5. **Begin the Discourse** — Go to the Discourse page, choose a cultivator, and start chatting!

## Concepts

### Elixir Pill (金丹)

> "The golden pill is the condensation of primordial unity."

An Elixir Pill is a **structured skill package of language patterns and persona traits**. Each pill contains:

| Field | Description |
|-------|-------------|
| identity_card | Identity card: the pill-persona's self-conception |
| expression_dna | Expression DNA: sentence length, formality, vocabulary, taboo words |
| mental_models | Mental models: thinking frameworks for approaching problems |
| decision_heuristics | Decision heuristics: expression strategies for specific situations |
| values / anti_patterns | Value orientation and anti-patterns |
| honest_limits | Honest limits: candid statements of capability boundaries |
| example_dialogues | Example dialogues: concrete demonstrations of style |

### Dao Cultivator (道人)

> "The Tao is that to which all things belong."

A Dao Cultivator is an **AI Agent**. Each cultivator has an independent base personality, system prompt, and chosen LLM. Their speech is shaped jointly by "base personality + consumed pills".

### Pill Consumption (服用金丹)

> "Consume the golden pill, shed your mortal bones, and comprehend all ages."

Binding a pill to a cultivator, with configurable **weight** and **sort_order**. One cultivator may consume multiple pills — the natures of many pills fused into one being.

### Skill-Persona Synthesis (金丹化性)

> "Gather herbs → Proportion → Into the furnace → Gentle nurture → Open the furnace and take the pill"

Mapping to the synthesis pipeline:

| Alchemy Step | Synthesis Step | Description |
|--------------|----------------|-------------|
| Gather herbs | Read traits | Load the cultivator's base personality and consumed pills |
| Proportion | Structural merge | Blend expression DNA and mental models by weight; deduplicate; detect conflicts |
| Into the furnace | Emergent inference | One LLM call distills the fused "elixir nature" and emergent rules |
| Gentle nurture | Caching | The synthesized system prompt is cached on the cultivator and rebuilt on change |
| Open the furnace | Discourse | Call the LLM with the synthesized prompt to generate replies |

## Screenshots

<p align="center">
  <i>🖼️ Pill Chamber — Elixir Pill (skill package) management</i>
</p>

<p align="center">
  <i>🖼️ Cultivator Hall — AI Agent management</i>
</p>

<p align="center">
  <i>🖼️ Discourse — streaming chat</i>
</p>

## API

See [docs/api.md](docs/api.md) for details.

### Quick Examples

```bash
# Create a pill (structured skill package)
curl -X POST http://localhost:8080/api/v1/pills \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Classical Chinese Pill",
    "description": "Makes the cultivator speak in classical Chinese",
    "skill_schema": {
      "identity_card": "I am an ancient scholar well-versed in the classics, fond of classical prose.",
      "expression_dna": {
        "sentence_length": "medium",
        "formality": 0.9,
        "vocabulary": ["之", "乎", "者", "也"],
        "taboo_words": ["你", "我", "的", "了"]
      }
    },
    "tags": ["classical", "elegant"],
    "author": "system",
    "version": "1.0.0"
  }'

# Create a cultivator
curl -X POST http://localhost:8080/api/v1/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Taishang Laojun",
    "personality": "You are Taishang Laojun, the patriarch of Taoism. Your speech is filled with supreme wisdom, and you expound all things through the philosophy of the Tao Te Ching.",
    "model_name": "gpt-4o"
  }'

# Consume a pill (with weight and order)
curl -X POST http://localhost:8080/api/v1/agents/1/pills \
  -H "Content-Type: application/json" \
  -d '{"pill_id": 1, "weight": 1.0, "sort_order": 0}'

# Create a session and chat (WebSocket)
# See the API docs and specs/001-skill-persona-alchemy-pivot/quickstart.md
```

## Documentation

| Document | Description |
|----------|-------------|
| [docs/architecture.md](docs/architecture.md) | System architecture deep dive |
| [docs/api.md](docs/api.md) | API reference |
| [docs/frontend.md](docs/frontend.md) | Frontend development guide |
| [docs/deployment.md](docs/deployment.md) | Deployment guide |
| [specs/001-skill-persona-alchemy-pivot/quickstart.md](specs/001-skill-persona-alchemy-pivot/quickstart.md) | Skill-Persona system quick verification guide |

## Project Structure

```
alchemy-furnace/
├── README.md                    # Project overview (Chinese)
├── README_EN.md                 # Project overview (English)
├── docker-compose.yml           # Docker orchestration
├── .env.example                 # Environment variable template
├── Makefile                     # Common commands
├── docs/                        # Maintenance docs
│   ├── architecture.md          # Architecture
│   ├── api.md                   # API reference
│   ├── frontend.md              # Frontend guide
│   └── deployment.md            # Deployment guide
├── backend/
│   ├── go/                      # Go API gateway
│   │   ├── cmd/server/
│   │   ├── dao/
│   │   ├── handler/
│   │   ├── model/
│   │   ├── pkg/
│   │   ├── service/
│   │   ├── go.mod
│   │   └── Dockerfile
│   └── python/                  # Python language-pattern synthesis engine
│       ├── app/
│       │   ├── api/
│       │   ├── core/
│       │   ├── models/
│       │   └── services/
│       ├── requirements.txt
│       └── Dockerfile
└── frontend/                    # React frontend
    ├── src/
    ├── public/
    ├── package.json
    └── Dockerfile
```

## Tech Stack

### Frontend
- React 18 + TypeScript
- Tailwind CSS + shadcn/ui
- Vite
- HashRouter (compatible with static deployment)
- WebSocket client
- React Context API state management

### Backend
- Go 1.21 + Gin
- GORM + PostgreSQL
- WebSocket
- RESTful API
- Zap logging

### Language-Pattern Synthesis Engine
- Python 3.11 + FastAPI
- OpenAI SDK (compatible with any OpenAI-style API)
- Structural merge + LLM emergent inference

## Roadmap

- [x] Pill management (skill package CRUD)
- [x] Cultivator management (Agent CRUD)
- [x] Pill consumption (weight and order configuration)
- [x] Language-pattern synthesis engine (structural merge + LLM emergent inference)
- [x] Pill trial preview (quick verification with temporary combinations)
- [x] Streaming chat (WebSocket)
- [ ] Multi-turn conversation context optimization
- [ ] Visualized elixir-clash warnings
- [ ] Multi-user access control
- [ ] Visualization of the refinement process
- [ ] Pill sharing and marketplace

## License

This project uses a custom license — see the [LICENSE](LICENSE) file for details.

**In brief:**

- **Personal use, learning, and research**: free to use, modify, and distribute
- **Commercial use**: you must retain the project name "炼丹炉 (Alchemy Furnace)" and developer attribution "yusanwen-code", and contact the author for a commercial license

---

<p align="center">
  <i>The Tao follows nature &middot; Refining elixir, forging intelligence</i>
</p>

<p align="center">
  Made with ☯️ by <a href="https://github.com/yusanwen-code">yusanwen-code</a>
</p>
