# Contributing to Rupee OS

Thanks for helping improve an India-first, local-AI personal finance tracker.

## Ground rules

- Be respectful — see [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).
- Prefer small, focused PRs over large dumps.
- Never commit real bank statements, passwords, or `.env` files.
- Statement data must stay local — do not add cloud LLM providers.

## Development setup

```bash
cp .env.example .env
docker compose up --build
./scripts/pull-ollama-model.sh   # once
```

| Service  | URL                      |
|----------|--------------------------|
| Frontend | http://localhost:3000    |
| API      | http://localhost:8080    |

Backend unit tests:

```bash
cd backend && go test ./...
```

Optional: run API/worker against local Postgres/Redis without Docker if you already have them — match `.env.example`.

## How to contribute

1. Open an [Issue](https://github.com/Omjaiswal26/rupee-os/issues) (or pick an existing one) before large work.
2. Fork + branch from `main` (`feat/…`, `fix/…`, `docs/…`).
3. Make changes; keep commits clear.
4. Add/adjust tests when touching parser, classification, or money math.
5. Open a Pull Request describing **what** and **why**.

### Good first areas

- Merchant keyword coverage (`backend/internal/classification/merchants.go`)
- Bank narration / column aliases in the parser
- Payee extraction edge cases + tests
- Docs and sample statements for more banks
- UI polish (accessibility, empty states)

### Synthetic data only

Use or extend files under `backend/testdata/`. Do not upload personal statements to Issues or PRs.

## Pull request checklist

- [ ] No secrets / real financial data
- [ ] `go test ./...` passes for backend changes
- [ ] Docs updated if behavior or setup changed
- [ ] Linked to an Issue when relevant

## Questions

Open a Discussion or Issue. For security-sensitive reports, avoid attaching statement files.
