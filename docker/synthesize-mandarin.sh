#!/usr/bin/env bash
set -euo pipefail
export OMP_NUM_THREADS=2
root=/opt/wirepod-mandarin
output=$(mktemp /tmp/vector-mandarin.XXXXXXXX.wav)
trap 'rm -f -- "$output"' EXIT
"$root/piper/piper" --model "$root/zh_CN-huayan-medium.onnx" --config "$root/zh_CN-huayan-medium.onnx.json" --length_scale 1.05 --sentence_silence 0.16 --output_file "$output" >&2
cat -- "$output"
