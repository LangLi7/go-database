#!/usr/bin/env bash
# go-database Model Downloader
# Lädt die 3 empfohlenen Modelle für lokale Nutzung.
# Erfordert: huggingface-cli (pip install huggingface-hub)
#
# Nutzung:
#   ./scripts/download-models.sh           → Alle 3 Modelle
#   ./scripts/download-models.sh ornith9   → Nur Ornith 1.0 9B
#   ./scripts/download-models.sh ds14      → Nur DeepSeek 14B
#   ./scripts/download-models.sh ornith35  → Nur Ornith 1.0 35B

set -euo pipefail
MODELS_DIR="${HOME}/.lmstudio/models"

download() {
  local publisher="$1" repo="$2" file="$3"
  local dir="${MODELS_DIR}/${publisher}/${repo}"
  mkdir -p "$dir"
  echo "→ Download ${file} ..."
  huggingface-cli download "${publisher}/${repo}" "$file" --local-dir "$dir"
  echo "  ✓ ${file} → ${dir}"
}

case "${1:-all}" in
  all)
    download deepreinforce-ai Ornith-1.0-9B-GGUF ornith-1.0-9b-q4_k_m.gguf
    download lmstudio-community DeepSeek-R1-Distill-Qwen-14B-GGUF DeepSeek-R1-Distill-Qwen-14B-Q4_K_M.gguf
    download deepreinforce-ai Ornith-1.0-35B-GGUF ornith-1.0-35b-q4_k_m.gguf
    ;;
  ornith9)
    download deepreinforce-ai Ornith-1.0-9B-GGUF ornith-1.0-9b-q4_k_m.gguf
    ;;
  ds14)
    download lmstudio-community DeepSeek-R1-Distill-Qwen-14B-GGUF DeepSeek-R1-Distill-Qwen-14B-Q4_K_M.gguf
    ;;
  ornith35)
    download deepreinforce-ai Ornith-1.0-35B-GGUF ornith-1.0-35b-q4_k_m.gguf
    ;;
  *)
    echo "Nutze: $0 [all|ornith9|ds14|ornith35]"
    exit 1
    ;;
esac
echo ""
echo "✓ Fertig. Starte LM Studio neu, um die Modelle zu laden."
