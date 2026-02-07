#!/usr/bin/env sh
# vim: set filetype=sh tabstop=2 shiftwidth=2 expandtab :
# shellcheck shell=sh
"${sourced_dadbb5b-false}" && return 0; sourced_dadbb5b=true

. ./utils.lib.sh

# Run tests.
task_test() {
  test $# = 0 && set -- ./...
  go test "$@"
}

# Run cmd.
task_run() {
  register_temp_cleanup
  package=./cmd/tblcalc/
  local a_out="$TEMP_DIR/a.out$exe_ext"
  go build -gcflags='all=-N -l' -tags=debug,nop -o "$a_out" "$package"
  "$a_out" "$@"
}

# Update documentation files.
task_doc() {
  mdpp --in-place --allow-remote \
    DEVELOPMENT.md \
    CLAUDE.md \
    #nop
}
