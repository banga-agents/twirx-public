#!/bin/sh

set -eu

usage() {
	printf '%s\n' 'usage: scripts/export-public-source.sh DESTINATION [REVISION]' >&2
	exit 2
}

[ "$#" -ge 1 ] && [ "$#" -le 2 ] || usage

destination=$1
revision=${2:-HEAD}

case "$destination" in
	''|/|.|..|~|"$HOME")
		printf '%s\n' 'refusing unsafe destination' >&2
		exit 2
		;;
esac

[ ! -e "$destination" ] || {
	printf 'destination already exists: %s\n' "$destination" >&2
	exit 2
}

repository=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
git -C "$repository" cat-file -e "$revision^{commit}"
source_revision=$(git -C "$repository" rev-parse "$revision^{commit}")

staging=$(mktemp -d "${TMPDIR:-/tmp}/twirx-public-export.XXXXXX")
cleanup() {
	rm -rf -- "$staging"
}
trap cleanup EXIT HUP INT TERM

git -C "$repository" archive "$source_revision" | tar -x -C "$staging"

# These patterns contain exact third-party archive bytes. The public export
# intentionally retains their TWIRX-authored manifests and digests.
for path in \
	"$staging"/atlas/archive-acquisitions/*/captures/*/representation.body \
	"$staging"/atlas/archive-acquisitions/*/captures/*/warc-record.gz \
	"$staging"/atlas/archive-acquisitions/*/raw/index-*.jsonl \
	"$staging"/atlas/archive-acquisitions/*/raw/range-*.warc.gz \
	"$staging"/reports/futo-acquisition-attempts/*/captures/*/representation.body \
	"$staging"/reports/futo-acquisition-attempts/*/captures/*/warc-record.gz \
	"$staging"/reports/futo-acquisition-attempts/*/raw/index-*.jsonl \
	"$staging"/reports/futo-acquisition-attempts/*/raw/range-*.warc.gz
do
	[ -e "$path" ] || continue
	rm -- "$path"
done

remaining=$(find "$staging" -type f \( \
	-name representation.body -o \
	-name warc-record.gz -o \
	-name 'range-*.warc.gz' -o \
	-name 'index-*.jsonl' \
	\) -print)
[ -z "$remaining" ] || {
	printf '%s\n' 'refusing export: a raw archive artifact remains:' >&2
	printf '%s\n' "$remaining" >&2
	exit 1
}

printf '%s\n' "$source_revision" >"$staging/PUBLIC_EXPORT_SOURCE.txt"
(
	cd "$staging"
	find . -type f ! -name PUBLIC_EXPORT_MANIFEST.sha256 -print0 \
		| LC_ALL=C sort -z \
		| xargs -0 sha256sum
) >"$staging/PUBLIC_EXPORT_MANIFEST.sha256"

destination_parent=$(dirname -- "$destination")
[ -d "$destination_parent" ] || {
	printf 'destination parent does not exist: %s\n' "$destination_parent" >&2
	exit 2
}

mv -- "$staging" "$destination"
trap - EXIT HUP INT TERM

printf 'public source export: %s\n' "$destination"
printf 'source revision: %s\n' "$source_revision"
printf 'manifest entries: %s\n' "$(wc -l <"$destination/PUBLIC_EXPORT_MANIFEST.sha256")"
