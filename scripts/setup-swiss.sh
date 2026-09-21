#!/bin/sh
set -eu
root=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
dest="$root/.deps/swisseph"
commit=56351c57e33916651a1aa32cfc6225e05c0ad865
mkdir -p "$root/.deps"
if [ ! -d "$dest/.git" ]; then
  git clone --filter=blob:none --no-checkout https://github.com/aloistr/swisseph.git "$dest"
fi
git -C "$dest" fetch --depth 1 origin "$commit"
git -C "$dest" checkout --detach "$commit"
test "$(git -C "$dest" rev-parse HEAD)" = "$commit"
make -C "$dest" clean
make -C "$dest" CFLAGS='-O2 -Wall -fPIC' libswe.a swetest
for file in sepl_18.se1 semo_18.se1 seas_18.se1; do
  test -s "$dest/ephe/$file"
done
(cd "$dest/ephe" && sha256sum sepl_18.se1 semo_18.se1 seas_18.se1) > "$root/.deps/ephemeris.sha256"
