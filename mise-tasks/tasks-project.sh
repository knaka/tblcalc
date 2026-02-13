# vim: set filetype=sh tabstop=2 shiftwidth=2 expandtab :
# shellcheck shell=sh
"${sourced_dadbb5b-false}" && return 0; sourced_dadbb5b=true

set -- "$PWD" "${0%/*}" "$@"; test -z "${_APPDIR-}" && { test "$2" = "$0" && _APPDIR=. || _APPDIR="$2"; cd "$_APPDIR" || exit 1; }
set -- _LIBDIR .lib "$@"
. ./.lib/utils.lib.sh
. ./.lib/tools.lib.sh
shift 2
cd "$1" || exit 1; shift 2

# Run tests.
task_test() {
  test $# = 0 && set -- ./...
  go test "$@"
}

# Run cmd.
task_run() {
  register_temp_cleanup
  local package="$PROJECT_DIR"/cmd/tblcalc/
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

# Run go cmd.
task_go() {
  go "$@"
}
