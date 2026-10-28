#!/usr/bin/env sh
# Pull the local classification model into the ollama volume (one-time / after wipe).
set -e
MODEL="${OLLAMA_MODEL:-qwen2.5:3b}"
echo "Pulling ${MODEL} via ollama service..."
docker compose exec ollama ollama pull "${MODEL}"
echo "Done. Model persists in the ollama_data volume."
