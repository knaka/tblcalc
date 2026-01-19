# vim: set filetype=sh tabstop=2 shiftwidth=2 expandtab :
# shellcheck shell=sh
"${sourced_7ecf25d-false}" && return 0; sourced_7ecf25d=true

. ./task.sh
. ./go.lib.sh
. ./mdpp.lib.sh

# Run Go tests.
subcmd_test() {
  go test "$@"
}

# Run application with debug information.
subcmd_run() {
  local package=./cmd/tblcalc/
  local a_out="$TEMP_DIR/a.out$exe_ext"
  go build -gcflags='all=-N -l' -tags=debug,nop -o "$a_out" "$package"
  "$a_out" "$@"
}

# Run lint
subcmd_lint() {
  "$HOME"/go/bin/golangci-lint-v2 run "$@"
}

# Update documentation files.
task_doc() {
  mdpp --in-place --allow-remote \
    DEVELOPMENT.md \
    CLAUDE.md \
    #nop
}
