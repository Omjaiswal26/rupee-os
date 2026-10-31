# Seed GitHub Issues (after repo is public)

`gh` isn’t authenticated in this environment. Create these Issues manually (or run after `gh auth login`):

```bash
gh issue create --title "Payee extraction: more Indian bank narration edge cases" --body "$(cat <<'EOF'
## Problem
Rail tokens (DEP/TFR) and NEFT/UPI narrations still produce weak merchants on some banks.

## Proposal
Expand ExtractPayeeCandidate coverage + golden tests from anonymized patterns (no real statements in the repo).

## Area
Classification / payee
EOF
)"

gh issue create --title "Merchant keyword pack for common India apps" --body "$(cat <<'EOF'
## Problem
Many UPI merchants fall through to Ollama or Other.

## Proposal
Curated keyword list (Food, Transport, Bills, Investments) with tests.

## Area
Classification / merchants
EOF
)"

gh issue create --title "Broader bank CSV/XLSX column aliases" --body "$(cat <<'EOF'
## Problem
Header detection fails on some bank exports.

## Proposal
Document aliases; add synthetic fixtures per bank family; improve diagnostics.

## Area
Parser
EOF
)"

gh issue create --title "Month health: MoM deltas + stale after corrections" --body "$(cat <<'EOF'
## Problem
Health is single-month snapshots; regenerating after categorize is manual.

## Proposal
Compare to previous month; mark report stale when classifications change.

## Area
Health
EOF
)"

gh issue create --title "DX: Ollama model preflight before classify queue" --body "$(cat <<'EOF'
## Problem
Missing model → rows stay pending with little UI signal.

## Proposal
API/worker preflight; clear error on import or health endpoint.

## Area
DX / worker
EOF
)"
```
