# Rupee OS

Open-source, single-user, India-first AI financial health tracker.

Upload a bank statement (**XLSX or CSV**) → validate **one calendar month** → store transactions → classify locally with Ollama (Qwen) → monthly dashboard.

## Stack

| Layer | Tech |
|-------|------|
| Frontend | Next.js + TypeScript |
| API | Go + Gin + GORM |
| Worker | Go + Asynq (same image as API) |
| DB | PostgreSQL |
| Queue | Redis |
| AI | Ollama + Qwen2.5 3B (**local only**) |
| Deploy | Docker Compose |

## Layout

```
frontend/                 Next.js app
backend/cmd/api/          HTTP server
backend/cmd/worker/       Asynq worker
backend/internal/         shared logic (handlers, services, parser, …)
backend/migrations/
docker-compose.yml
```

## Quick start

```bash
cp .env.example .env
docker compose up --build
```

| Service | URL |
|---------|-----|
| Frontend | http://localhost:3000 |
| API | http://localhost:8080/health |
| Ollama | http://localhost:11434 |

### Pull the model (once)

```bash
./scripts/pull-ollama-model.sh
```

Models persist in the `ollama_data` volume.

## Statement rules

- Formats: **`.xlsx` / `.csv`** (Indian bank exports with Date + Narration + Debit/Credit columns).
- Each upload must contain transactions for **exactly one calendar month** (validated from transaction dates).
- Multi-month, empty, or undated statements are rejected with a clear error.
- Duplicate files (same SHA-256) and duplicate transactions are skipped.
- Statement contents are never sent to external AI APIs.

## API (V1)

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/health` | health |
| `POST` | `/imports` | multipart `file` upload |
| `GET` | `/imports` | list imports |
| `GET` | `/imports/:id` | import status / errors |
| `GET` | `/months` | available months |
| `GET` | `/months/:yyyy-mm/summary` | income / expense / net / categories |
| `GET` | `/months/:yyyy-mm/transactions` | `?q=&category=&status=&limit=&offset=` |
| `GET` | `/transactions/review` | needs review |
| `PATCH` | `/transactions/:id/classification` | correct + optional `create_rule` |
| `GET` | `/rules` | learned rules |
| `GET` | `/meta/categories` | allowed categories/types |

Example upload:

```bash
curl -F file=@backend/testdata/sample_single_month.csv http://localhost:8080/imports
```

## Local backend tests

```bash
cd backend
go test ./...
```

## Volumes

- `postgres_data`, `redis_data`, `ollama_data`, `uploads_data`

## Troubleshooting

- **API unhealthy:** `docker compose logs api` — wait for Postgres/Redis.
- **Import stuck queued:** ensure worker is up (`docker compose logs worker`).
- **Classification empty:** pull the model with `./scripts/pull-ollama-model.sh`.
- **MULTI_MONTH error:** export a single calendar month from your bank.
- **Reset:** `docker compose down -v` then `up --build` and re-pull the model.
