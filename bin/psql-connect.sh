#!/bin/bash
set -e
set -u

source .env
psql "${DB_URL}"
