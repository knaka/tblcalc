#!/usr/bin/env bash
set -- _562af7b "$@"; eval "shift; \${$1-false} || ! $1=true" && return # shpp:source_guard

pushd "${BASH_SOURCE[0]%[/\\]*}" &>/dev/null || pushd . >/dev/null
. ../.lib/utils.sh
popd >/dev/null || exit

# Run tests.
task_test() {
  test $# = 0 && set -- ./...
  go test "$@"
}

# Run cmd.
task_run() {
  register_temp_cleanup
  local package="$PROJECT_DIR"/cmd/tblcalc/
  local a_out="$TEMP_DIR/a.out$EXE_EXT"
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

# Run go cmd.
task_go() {
  go "$@"
}
