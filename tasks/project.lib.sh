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

# Releases · golangci/golangci-lint https://github.com/golangci/golangci-lint/releases
golangci_lint_version_a6d4fc4="2.8.0"

golangci_lint() {
  # shellcheck disable=SC2016
  run_fetched_cmd \
    --name="golangci-lint" \
    --ver="$golangci_lint_version_a6d4fc4" \
    --os-map="$goos_map" \
    --arch-map="$goarch_map" \
    --ext-map="$archive_ext_map" \
    --url-template='https://github.com/golangci/golangci-lint/releases/download/v${ver}/golangci-lint-${ver}-${os}-${arch}${ext}' \
    --rel-dir-template='golangci-lint-${ver}-${os}-${arch}' \
    -- \
    "$@"
}

# Run lint
subcmd_lint() {
  golangci_lint run "$@"
}

# Update documentation files.
task_doc() {
  mdpp --in-place --allow-remote \
    DEVELOPMENT.md \
    CLAUDE.md \
    #nop
}
