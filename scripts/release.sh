#!/usr/bin/env bash
set -euo pipefail

version="${1:-}"
if [[ -z "$version" ]]; then
  echo "usage: $0 VERSION" >&2
  exit 2
fi

if [[ ! "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[A-Za-z0-9.-]+)?$ ]]; then
  echo "invalid version: $version" >&2
  exit 2
fi

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

python3 - "$version" <<'PY'
from pathlib import Path
import sys

version = sys.argv[1]
path = Path("internal/version/version.go")
text = path.read_text()
start = 'const Version = "'
line = next(line for line in text.splitlines() if line.startswith(start))
old = line[len(start):-1]
path.write_text(text.replace(line, f'{start}{version}"\n'))
Path("VERSION").write_text(version + "\n")
print(f"version: {old} -> {version}")
PY

gofmt -w internal/version/version.go
go test ./...
go build -o bin/hypr-dock ./cmd/hypr-dock
go build -o bin/hypr-dock-settings ./cmd/hypr-dock-settings
go build -o bin/hypr-dockctl ./cmd/hypr-dockctl
go build -o bin/hypr-alttab ./cmd/hypr-alttab

git add -A
git add -f bin/hypr-dock bin/hypr-dock-settings bin/hypr-dockctl bin/hypr-alttab
git commit -m "release: HyprDock+ $version"
git push origin HEAD

echo "released HyprDock+ $version"
