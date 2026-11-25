#!/usr/bin/env bash
set -euo pipefail

root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
source=https://raw.githubusercontent.com/kercre123/wire-pod/a04190f93491435257abc2a2d933f7190b3a6f3a
cd "$root"

fetch_compatibility_file() {
    local path="$1" hash="$2"
    local staging
    staging="$(mktemp "${path}.XXXXXXXX")"
    if curl -fsSL --retry 2 --max-time 30 "$source/$path" -o "$staging" &&
        printf '%s  %s\n' "$hash" "$staging" | sha256sum -c -; then
        chmod 600 "$staging"
        mv -f -- "$staging" "$path"
    else
        rm -f -- "$staging"
        return 1
    fi
}

# These two publicly distributed files are used by stock EP firmware.
fetch_compatibility_file chipper/epod/ep.key 35dc195c7267ac48de07872347a3a8fb3f9068ae21c9d18b51fa1a13b614c35b
fetch_compatibility_file chipper/pkg/wirepod/setup/robot_sshkey e0591141a97bfd821360c06f946c01ed43771a222fff14c58a3561b43922005a
