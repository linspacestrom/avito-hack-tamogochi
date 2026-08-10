#!/bin/sh
set -eu

# Migration files use a contiguous sequence and include matching down scripts.

: "${MIGRATIONS_PATH:=/migrations}"
: "${MIGRATE_DATABASE_URL:?MIGRATE_DATABASE_URL must be set}"

set -- "$MIGRATIONS_PATH"/*.up.sql
if [ ! -e "$1" ]; then
    echo "no up migrations found in $MIGRATIONS_PATH" >&2
    exit 1
fi

expected=1
for up_path in "$@"; do
    filename=${up_path##*/}
    version=${filename%%_*}

    case "$version" in
        ''|*[!0-9]*)
            echo "invalid migration filename: $filename" >&2
            exit 1
            ;;
    esac

    version_number=$(printf '%s' "$version" | sed 's/^0*//')
    : "${version_number:=0}"
    if [ "$version_number" -ne "$expected" ]; then
        printf 'migration sequence mismatch: expected %06d, found %s\n' \
            "$expected" "$version" >&2
        exit 1
    fi

    down_path=${up_path%.up.sql}.down.sql
    if [ ! -f "$down_path" ]; then
        echo "missing down migration for $filename" >&2
        exit 1
    fi

    expected=$((expected + 1))
done

exec migrate \
    -path="$MIGRATIONS_PATH" \
    -database="$MIGRATE_DATABASE_URL" \
    up
