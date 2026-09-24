#!/bin/sh
# Package only tracked source and locked dependencies, never working-tree secrets.
set -eu
root=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
cd "$root"
git diff --exit-code HEAD --
revision=$(git rev-parse HEAD)
scratch=$(mktemp -d)
mkdir -p "$scratch/kripa/.deps/swisseph" "$root/bin"
git archive HEAD | tar -x -C "$scratch/kripa"
git -C .deps/swisseph archive 56351c57e33916651a1aa32cfc6225e05c0ad865 | tar -x --exclude=ephe -C "$scratch/kripa/.deps/swisseph"
# All native source/notices, but only the data files actually shipped in the image.
mkdir -p "$scratch/kripa/.deps/swisseph/ephe"
for file in sepl_18.se1 semo_18.se1 seas_18.se1; do
  cp ".deps/swisseph/ephe/$file" "$scratch/kripa/.deps/swisseph/ephe/$file"
done
(cd "$scratch/kripa" && go mod vendor)
tar -czf "$root/bin/kripa-source-$revision.tar.gz" -C "$scratch" kripa
# Keep the temporary extracted bundle available for inspection; no broad cleanup.
printf 'Source bundle: bin/kripa-source-%s.tar.gz\n' "$revision"
