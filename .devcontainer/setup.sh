#!/usr/bin/env bash

SCRIPTPATH="$(dirname "$(realpath "$0")")"

mkdir -p ~/.local/share/opencode
mkdir -p ~/.local/state/opencode
mkdir -p ~/.omp
mkdir -p ~/.config/gh/

ENVFILE_PATH="$SCRIPTPATH/../.env"
if [ ! -f "$ENVFILE_PATH" ]
then
	touch "$ENVFILE_PATH"
fi
