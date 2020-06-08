#!/bin/bash
echo "=== INSTALLING POSTGRES ==="
sleep 1

set -e
set -u

# Notes on convention:
# variables expected to be imported or exported are ALL_CAPS and prefixed with POSTGRES_
# variables expected to remain private are lowercase and prefixed as to not be affected by `source`

# source .env
DOWNLOADS_DIR=${DOWNLOADS_DIR:-"$HOME/Downloads"}
OPT_DIR=${OPT_DIR:-"$HOME/Applications"}
DATA_DIR=${DATA_DIR:-"$HOME/.local/share"}
POSTGRES_DATA_DIR=${POSTGRES_DATA_DIR:-"$DATA_DIR/postgres/var"}
mkdir -p "$DOWNLOADS_DIR"
mkdir -p "$OPT_DIR"
mkdir -p "$POSTGRES_DATA_DIR"

is_macos="$(uname -a | grep -i darwin)"
if [ -n "$is_macos" ]; then
  TRASH_DIR=${TRASH_DIR:-"$HOME/.Trash"}
  POSTGRES_VERSION=${POSTGRES_VERSION:-"10.13"} # 10.13-1
  POSTGRES_BUILD=${POSTGRES_BUILD:-"1-osx"}
  postgres_pkg="postgresql-${POSTGRES_VERSION}-${POSTGRES_BUILD}-binaries.zip"
  is_zip="true"
else
  TRASH_DIR=${TRASH_DIR:-"$HOME/tmp"}
  POSTGRES_VERSION=${POSTGRES_VERSION:-"10.12"} # 10.12-1
  POSTGRES_BUILD=${POSTGRES_BUILD:-"1-linux-x64"}
  postgres_pkg="postgresql-${POSTGRES_VERSION}-${POSTGRES_BUILD}-binaries.tar.gz"
  is_zip=""
fi

mkdir -p "$TRASH_DIR"

# https://www.enterprisedb.com/download-postgresql-binaries

postgres_tmp="$(mktemp -d -t postgres-installer.XXXXXXXX)"
postgres_unpack="pgsql"
postgres_dir="postgres-server-${POSTGRES_VERSION}"
postgres_lnk="postgres-server"

echo "Here's what this script will do:"
echo "    • Download postgres server v${POSTGRES_VERSION}"
echo "    • Install it to ${OPT_DIR}/${postgres_dir}"
echo "    • Link that to ${OPT_DIR}/${postgres_lnk}"
echo "    • Create a database in $POSTGRES_DATA_DIR (first-time only)"
echo "    • Start Postgres with $OPT_DIR/${postgres_lnk}/bin/pg_ctl"

echo ""
echo "Working directory is ${postgres_tmp}"
echo ""
if [ -f "${DOWNLOADS_DIR}/${postgres_pkg}" ]; then
  rsync -aq "${DOWNLOADS_DIR}/${postgres_pkg}" "$postgres_tmp/$postgres_pkg"
else
  echo "Downloading $postgres_pkg"
  curl -fSL --progress-bar 'https://get.enterprisedb.com/postgresql/'"${postgres_pkg}"'?ls=Crossover&type=Crossover' -o "$postgres_tmp/$postgres_pkg"
  rsync -aq "$postgres_tmp/$postgres_pkg" "${DOWNLOADS_DIR}/"
fi
pushd "$postgres_tmp" >/dev/null
  if [ -n "$is_zip" ]; then
    unzip -q "$postgres_pkg"
  else
    tar xvf "$postgres_pkg"
  fi
  mv "$postgres_unpack" "$postgres_dir"
popd >/dev/null
if [ -d "$OPT_DIR/$postgres_dir" ]; then
  mv "$OPT_DIR/$postgres_dir" "$TRASH_DIR/$postgres_dir".$(date '+%Y-%m-%d_%H-%M-%S' )
  echo "moved old $OPT_DIR/$postgres_dir to the Trash folder"
fi
mv "$postgres_tmp/$postgres_dir" "$OPT_DIR/"
rm -f "$OPT_DIR/$postgres_lnk"
ln -s "$OPT_DIR/$postgres_dir" "$OPT_DIR/$postgres_lnk"

echo "postgres" > "${postgres_tmp}/pwfile"

mkdir -p "$POSTGRES_DATA_DIR"
chmod 0700 "$POSTGRES_DATA_DIR"
if [ ! -f "$POSTGRES_DATA_DIR/postgresql.conf" ]; then
  "$OPT_DIR/$postgres_lnk/bin/initdb" \
    -D "$POSTGRES_DATA_DIR/" \
    --username postgres --pwfile "${postgres_tmp}/pwfile" \
    --auth-local=password --auth-host=password
fi

echo "PostgreSQL installed, database initialized in $POSTGRES_DATA_DIR/"

rm -rf "${postgres_tmp}"
