#!/bin/sh
# Run from the module root with the same target environment as the binaries.
set -eu
output=${1:?usage: collect-go-notices.sh OUTPUT_DIRECTORY}
mkdir -p "$output/stdlib"
goroot=$(go env GOROOT)
printf '%s\n' "$(go env GOVERSION)" > "$output/stdlib/VERSION"

packages=$(mktemp)
trap 'rm -f "$packages"' EXIT
go list -deps -f '{{if .Standard}}stdlib||{{.Dir}}{{else if and .Module (not .Module.Main)}}{{.Module.Path}}@{{.Module.Version}}|{{.Module.Dir}}|{{.Dir}}{{end}}' ./cmd/... > "$packages"
while IFS='|' read -r module root dir; do
    [ -n "$module" ] || continue
    if [ "$module" = stdlib ]; then root=$goroot; fi
    # Nested packages can carry additional terms, for example zstd's xxhash.
    while :; do
        found=
        target="$output/$module${dir#"$root"}"
        for file in "$dir"/LICEN[CS]E* "$dir"/COPYING* "$dir"/NOTICE* "$dir"/AUTHORS "$dir"/PATENTS; do
            [ -f "$file" ] || continue
            found=yes
            if [ ! -f "$target/${file##*/}" ]; then
                mkdir -p "$target"
                cp "$file" "$target/"
            fi
        done
        [ "$dir" != "$root" ] || break
        dir=${dir%/*}
    done
    if [ -z "$found" ]; then
        printf 'Missing license for %s\n' "$module" >&2
        exit 1
    fi
done < "$packages"
