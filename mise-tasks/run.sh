#!/usr/bin/env sh
# vim: set filetype=sh tabstop=2 shiftwidth=2 expandtab :
# shellcheck shell=sh
"${sourced_dadbb5b-false}" && return 0; sourced_dadbb5b=true

#MISE description="Run cmd."

set -- "$PWD" "${0%/*}" "$@"; if test "$2" != "$0"; then cd "$2" 2>/dev/null || :; fi
. ./.lib.sh
  init_temp_dir
cd "$1"; shift 2

package=./cmd/tblcalc/
a_out="$TEMP_DIR/a.out$exe_ext"
go build -gcflags='all=-N -l' -tags=debug,nop -o "$a_out" "$package"
"$a_out" "$@"
