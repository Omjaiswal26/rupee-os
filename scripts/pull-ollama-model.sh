#!/usr/bin/env sh
# Pull local Ollama models into the ollama volume (one-time / after wipe).
set -e
MODEL="${OLLAMA_MODEL:-qwen2.5:3b}"
HEALTH_MODEL="${OLLAMA_HEALTH_MODEL:-qwen2.5:7b}"
echo "Pulling classify model ${MODEL}..."
docker compose exec ollama ollama pull "${MODEL}"
echo "Pulling health model ${HEALTH_MODEL}..."
docker compose exec ollama ollama pull "${HEALTH_MODEL}"
echo "Done. Models persist in the ollama_data volume."
