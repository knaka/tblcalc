# vim: set filetype=sh tabstop=2 shiftwidth=2 expandtab :
# shellcheck shell=sh
"${sourced_e8fb65d-false}" && return 0; sourced_e8fb65d=true

# ==========================================================================
#region Environment variables. If not set by the caller, they are set later in `tasksh_main`

# The original working directory when the script was started.
: "${ORIGINAL_CWD=}"
: "${ORIGINAL_CWD:=${MISE_ORIGINAL_CWD:-}}"
: "${ORIGINAL_CWD:=$PWD}"
# Aliases
: "${ORIGINAL_PWD:=${ORIGINAL_CWD}}"
: "${INITIAL_PWD:=${ORIGINAL_CWD}}"

# For platforms other than Windows
: "${LOCALAPPDATA:=/}"

#endregion

# ==========================================================================
#region Temporary directory and cleaning up

TEMP_DIR=; unset TEMP_DIR

# Create a temporary directory and assign $TEMP_DIR env var
register_temp_cleanup() {
  test "${TEMP_DIR+set}" = set && return 0
  TEMP_DIR="$(mktemp -d)"
  # shellcheck disable=SC2064
  trap "rm -fr '$TEMP_DIR'" EXIT
}

# Create a temporary directory and assign $TEMP_DIR env var. Obsolete: use register_temp_cleanup instead.
init_temp_dir() {
  test "${TEMP_DIR+set}" = set && return 0
  TEMP_DIR="$(mktemp -d)"
  # shellcheck disable=SC2064
  trap "rm -fr '$TEMP_DIR'" EXIT
}

readonly stmts_file_id=523f163

# Chain traps to avoid overwriting the previous trap.
chaintrap() {
  local stmts_new="$1"
  shift 
  init_temp_dir || return $?
  # Base path of the file containing the statements to be called during finalization
  local stmts_file_base="$TEMP_DIR"/"$stmts_file_id"
  local stmts_old_file="$TEMP_DIR"/347803f
  local sigspec
  for sigspec in "$@"
  do
    local stmts_file="$stmts_file_base"-"$sigspec"
    if test -f "$stmts_file"
    then
      cp "$stmts_file" "$stmts_old_file"
    else
      touch "$stmts_old_file"
    fi
    echo "{ $stmts_new; };" >"$stmts_file"
    cat "$stmts_old_file" >>"$stmts_file"
    # shellcheck disable=SC2064 # "Use single quotes, otherwise this expands now rather than when signalled."
    # shellcheck disable=SC2154 # "var is referenced but not assigned."
    trap "rc=\$?; test -r '$stmts_file' && . '$stmts_file'; rm -fr '$TEMP_DIR'; exit \$rc" "$sigspec"
  done
}

# Call the finalization function before `exec` which does not call trap function.
finalize() {
  test "${TEMP_DIR+set}" = set || return 0
  local stmts_file_base="$TEMP_DIR"/"$stmts_file_id"
  local stmts_file="$stmts_file_base"-EXIT
  # shellcheck disable=SC1090 # "Can't follow non-constant source. Use a directive to specify location"
  test -f "$stmts_file" && . "$stmts_file"
  rm -fr "$TEMP_DIR"
}

#endregion

# ==========================================================================
#region Utilities

# Guard against multiple calls. $1 is a unique ID
first_call() {
  eval "\${called_$1-false}" && return 1
  eval "called_$1=true"
}

# Check if stdout is tty.
is_terminal() {
  test -t 1
}

# Check if external command exists in $PATH.
has_external_command() {
  # test -x "$(command -v "$1" 2>/dev/null)"
  # command which "$1" >/dev/null # `command` does not ignore builtins
  # env which "$1" >/dev/null
  which "$1" >/dev/null
}

#endregion

# ==========================================================================
#region Platform detection. Detect platform without using subprocesses whenever possible.

is_linux() {
  test -d /proc -o -d /sys
}

is_macos() {
  test -r /System/Library/CoreServices/SystemVersion.plist
}

is_windows() {
  test -d "c:/" -a ! -d /proc
}

# Executable file extension.
exe_ext=
# shellcheck disable=SC2034
is_windows && exe_ext=".exe"

is_debian() {
  test -f /etc/debian_version
}

is_bsd() {
  is_macos || test -r /etc/rc.subr
}

is_alpine() {
  test -f /etc/alpine-release
}

#endregion

# ==========================================================================
#region Directory stack

psv_dirs_4c15d80=""

# `pushd` alternative.
push_dir() {
  local pwd="$PWD"
  if ! cd "$1"
  then
    echo "Directory does not exist: $1" >&2
    return 1
  fi
  psv_dirs_4c15d80="$pwd|$psv_dirs_4c15d80"
}

# `popd` alternative.
pop_dir() {
  if test -z "$psv_dirs_4c15d80"
  then
    echo "Directory stack is empty" >&2
    return 1
  fi
  local dir="${psv_dirs_4c15d80%%|*}"
  psv_dirs_4c15d80="${psv_dirs_4c15d80#*|}"
  cd "$dir" || return 1
}

#endregion

# ==========================================================================
#region Process management

windows_exe_extensions=".exe .EXE .cmd .CMD .bat .BAT"

# Invoke external command with proper executable extension, with the specified invocation mode.
#
# Invocation mode can be specified via INVOCATION_MODE environment variable:
#   INVOCATION_MODE=standard: (Default) Run the command in the current process.
#   INVOCATION_MODE=exec: Replace this process with the command.
#   INVOCATION_MODE=exec-direct: Replace this process with the command, without calling cleanups.
#   INVOCATION_MODE=background: Run the command in the background.
#
# Invocation mode can be specified also with `--invocation-mode=...` option. The option is excluded from final arguments which are passed to the external command.
#
# Command-specific invocation mode can be set using INVOCATION_MODE_<command> variables:
#   INVOCATION_MODE_foo=background: Run external command `foo` in the background.
#   The command name is extracted from the basename of the first argument, with Windows
#   executable extensions (.exe, .cmd, etc.) stripped if on Windows platform.
invoke() {
  if test $# -eq 0
  then
    echo "No command specified" >&2
    exit 1
  fi
  local invocation_mode="${INVOCATION_MODE:-standard}"
  local base="${1##*/}"
  if is_windows
  then
    local ext
    local IFS=" "
    for ext in $windows_exe_extensions
    do
      base="${base%"$ext"}"
    done
  fi
  if eval "test \"\${INVOCATION_MODE_$base+set}\" = set"
  then
    eval "invocation_mode=\"\${INVOCATION_MODE_$base}\""
  fi
  local arg
  for arg in "$@"
  do
    case "$arg" in
      (--invocation-mode=*)
        invocation_mode="${arg#--invocation-mode=}"
        ;;
      (*)
        set -- "$@" "$arg"
        ;;
    esac
    shift
  done
  local cmd="$1"
  case "$1" in
    (*/*)
      if is_windows
      then
        local ext
        for ext in $windows_exe_extensions
        do
          if test -x "$cmd$ext"
          then
            shift
            set -- "$cmd$ext" "$@"
            break
          fi
        done
      fi
      if ! test -x "$1"
      then
        echo "Command not found: $1" >&2
        exit 1
      fi
      ;;
    (*)
      if is_windows
      then
        local ext
        for ext in $windows_exe_extensions
        do
          if command -v "$1$ext" >/dev/null 2>&1
          then
            shift
            set -- "$cmd$ext" "$@"
            break
          fi
        done
      fi
      if ! command -v "$1" >/dev/null 2>&1
      then
        echo "Command not found: $1" >&2
        exit 1
      fi
      ;;
  esac
  "$VERBOSE" && echo "Launching $* in mode $invocation_mode, in $PWD." >&2
  case "$invocation_mode" in
    (exec)
      finalize
      # exec executes the external command even if a function with the same name is defined.
      exec "$@"
      ;;
    (exec-direct)
      exec "$@"
      ;;
    (background)
      command "$@" &
      ;;
    (standard)
      command "$@"
      ;;
    (*)
      echo "Unknown invocation mode: $invocation_mode" >&2
      exit 1
      ;;
  esac
}

# ==========================================================================
#region Misc

# Unit separator (US), Information Separator One (0x1F)
# shellcheck disable=SC2034
readonly us=""
# shellcheck disable=SC2034
readonly is="$us"
# shellcheck disable=SC2034
readonly is1="$us"

# Canonicalize path
canon_path() {
  local target="$1"
  target="$(echo "$target" | sed -E -e 's|[/\\]+|/|g')"
  if test -d "$target"
  then
    # -P: Handle the operand dot-dot physically
    (
      cd -P -- "$target" || exit 1
      echo "$PWD"
    )
  else
    (
      cd -P -- "$(dirname -- "$target")" || exit 1
      printf "%s/%s\n" "$PWD" "$(basename -- "$target")"
    )
  fi
}

# Check if root directory
is_root_dir() {
  local dir="$1"
  dir="$(canon_path "$dir")"
  local parent_dir
  parent_dir="$(dirname "$dir")"
  test "$dir" = "$parent_dir"
}

# Run command after globbing arguments. For Windows environment.s
glob_and_run() {
  local cmd="$1"
  shift
  local arg
  for arg in "$@"
  do
    shift
    case "$arg" in
      (-*) set -- "$@" "$arg" ;;
      (*\?*|*\**)
        if test -e "$arg"
        then
          set -- "$@" "$arg"
          continue
        fi
        arg="$(echo "$arg" | sed -e 's|\\|/|g')"
        for arg2 in $arg
        do
          set -- "$@" "$arg2"
        done
        ;;
      (*) set -- "$@" "$arg" ;;
    esac
  done
  command $cmd "$@"
}

# Wait for one or more servers to respond with HTTP 200. Checks each URL sequentially with a 60-second timeout per URL.
wait_for_server() {
  local url
  local max_attempts=60
  for url in "$@"
  do
    echo "Waiting for server at $url to be ready ..." >&2
    local attempts=0
    while :
    do
      if curl -s -o /dev/null -w "%{http_code}" "$url" 2>/dev/null | grep -q "200"
      then
        echo "✓ Server is ready at $url" >&2
        break
      fi
      attempts=$((attempts + 1))
      if test $attempts -ge $max_attempts
      then
        echo "✗ Server at $url did not respond with 200 after $max_attempts seconds" >&2
        return 1
      fi
      sleep 1
    done
  done
}

# Convenient for cleaning logs.
strip_escape_sequences() {
  # ANSI escape code - Wikipedia https://en.wikipedia.org/wiki/ANSI_escape_code
  # BusyBox sed(1) does not accept `\octal` or `\xhex`.
  sed -E -e 's/\[[0-9;]*[ABCDEFGHJKSTmin]//g'
}

# Absolute path to relative path
abs2rel() {
  local target="$1"
  shift
  local source="$PWD"
  if test "$#" -gt 0
  then
    source="$1"
  fi
  local common="$source"
  local back=
  while test "${target#"$common"}" = "${target}"
  do
    common=$(dirname "$common")
    back="../${back}"
  done
  echo "${back}""${target#"$common"/}"
}

# shuf(1) for macOS environment.
if ! command -v shuf >/dev/null 2>&1
then
  alias shuf='sort -R'
fi

is_macos && alias sha1sum='shasum -a 1'

# Memoize the (mainly external) command output.
memoize() {
  local cache_file_path
  cache_file_path="$TEMP_DIR"/cache-"$(echo "$@" | sha1sum | cut -d' ' -f1)"
  if ! test -r "$cache_file_path"
  then
    "$@" >"$cache_file_path" || return $?
  fi
  cat "$cache_file_path"
}

# Current cache file path for memoization.
cache_file_path_cb3727b=

begin_memoize() {
  cache_file_path_cb3727b="$TEMP_DIR"/cache-"$(echo "$@" | sha1sum | cut -d' ' -f1)"
  if test -r "$cache_file_path_cb3727b"
  then
    cat "$cache_file_path_cb3727b"
    return 1
  fi
  exec 9>&1
  exec >"$cache_file_path_cb3727b"
}

end_memoize() {
  exec 1>&9
  exec 9>&-
  cat "$cache_file_path_cb3727b"
}

# The path to the shell executable which is running the script.
shell_path() {
  begin_memoize d57754a "$@" || return 0

  if test "${BASH+set}" = set
  then
    echo "$BASH"
  elif is_windows && test "${SHELL+set}" = set && test "$SHELL" = "/bin/sh" && "$SHELL" --help 2>&1 | grep -q "BusyBox"
  then
    echo "$SHELL"
  else
    local path=
    if test -e /proc/$$/exe
    then
      path="$(realpath /proc/$$/exe)" || return 1
    else
      path="$(realpath "$(ps -p $$ -o comm=)")" || return 1
    fi 
    echo "$path"
  fi

  end_memoize
}

# The implementation name of the shell which is running the script. Not "sh" but "bash", "ash", "dash", etc.
shell_name() {
  if test "${BASH+set}" = set
  then
    echo "bash"
  elif is_windows && test "${SHELL+set}" = set && test "$SHELL" = "/bin/sh" && "$SHELL" --help 2>&1 | grep -q "BusyBox"
  then
    echo "ash"
  else
    local path=
    if test -e /proc/$$/exe
    then
      path="$(realpath /proc/$$/exe)" || return 1
    else
      path="$(realpath "$(ps -p $$ -o comm=)")" || return 1
    fi
    case "${path##*/}" in
      (bash) echo "bash";;
      (ash) echo "ash";;
      (dash) echo "dash";;
      (sh|busybox)
        if "$path" --help 2>&1 | grep -q "BusyBox"
        then
          echo "ash"
        else
          echo "Cannot detect shell: $path" >&2
          return 1
        fi
        ;;
      (*)
        echo "Unknown shell: $path" >&2
        return 1
        ;;
    esac
  fi
}

is_dash() {
  test "$(shell_name)" = "dash"
}

is_ash() {
  test "$(shell_name)" = "ash"
}

is_bash() {
  test "$(shell_name)" = "bash"
}

# Check if the file(s)/directories are newer than the destination.
newer() {
  local found_than=false
  local dest=
  local arg
  for arg in "$@"
  do
    shift
    if test "$arg" = "--than"
    then
      found_than=true
    elif $found_than
    then
      dest="$arg"
    else
      set -- "$@" "$arg"
    fi
  done
  if test -z "$dest"
  then
    echo "Missing --than option" >&2
    exit 1
  fi
  if test "$#" -eq 0
  then
    echo "No source files specified" >&2
    exit 1
  fi
  # If the destination does not exist, sources are considered newer than the destination.
  if ! test -e "$dest"
  then
    echo "Destination does not exist: $dest" >&2
    return 0
  fi
  # If the destination is a directory, the newest file in the directory is used.
  if test -d "$dest"
  then
    if is_bsd
    then
      dest="$(find "$dest" -type f -exec stat -l -t "%F %T" {} \+ | cut -d' ' -f6- | sort -n | tail -1 | cut -d' ' -f3)"
    else
      dest="$(find "$dest" -type f -exec stat -Lc '%Y %n' {} \+ | sort -n | tail -1 | cut -d' ' -f2)"
    fi
  fi
  if test -z "$dest"
  then
    echo "No destination file found" >&2
    return 0
  fi
  test -n "$(find "$@" -newer "$dest" 2>/dev/null)"
}

# Returns true if no source file is newer than the destination file.
older() {
  ! newer "$@"
}

# Kill child processes for each shell/platform.
kill_child_processes() {
  if is_windows
  then
    # Windows BusyBox ash
    # If the process is killed by PID, ash does not kill `exec`ed subprocesses.
    local jids
    jids="$TEMP_DIR"/jids
    # ash provides a "jobs pipe".
    jobs | sed -E -e 's/^[^0-9]*([0-9]+).*Running *(.*)/\1/' >"$jids"
    while read -r jid
    do
      kill "%$jid" || :
      wait "%$jid" || :
      echo "Killed %$jid" >&2
    done <"$jids"
  elif is_macos
  then
    pkill -P $$ || :
  elif is_linux
  then
    if is_bash
    then
      local jids
      jids="$TEMP_DIR"/jids
      # Bash provides a "jobs pipe".
      jobs | sed -E -e 's/^[^0-9]*([0-9]+).*Running *(.*)/\1/' >"$jids"
      while read -r jid
      do
        kill "%$jid" || :
        wait "%$jid" || :
        echo "Killed %$jid" >&2
      done <"$jids"
    else
      pkill -P $$ || :
    fi
  else
    echo "kill_child_processes: Unsupported platform or shell." >&2
    exit 1
  fi
}

# Register child-proceses cleanup trap handler.
register_child_cleanup() {
  chaintrap kill_child_processes EXIT TERM INT
}

# Obsolete: use `register_child_cleanup`.
defer_child_cleanup() {
  chaintrap kill_child_processes EXIT TERM INT
}

# Open the URL in the browser.
browse() {
  if is_linux
  then
    xdg-open "$1"
  elif is_macos
  then
    open "$1"
  elif is_windows
  then
    PowerShell -Command "Start-Process '$1'"
  else
    echo "Unsupported OS: $(uname -s)" >&2
    exit 1
  fi
}

# Get a key from the user without echoing.
get_key() {
  if is_linux || is_macos
  then
    local saved_stty; saved_stty="$(stty -g)" || return $?
    stty -icanon -echo
    dd bs=1 count=1 2>/dev/null
    stty "$saved_stty"
    return
  fi
  local key
  # Bash and BusyBox Ash provide the `-s` (silent mode) option.
  if is_ash || is_bash
  then
    # shellcheck disable=SC3045
    read -rsn1 key
  # Otherwise, the input is echoed
  else
    read -r key
  fi
  echo "$key"
}

# Show a message and get input from the user.
prompt() {
  local message="${1:-Text}"
  local default="${2:-}"
  printf "%s: (%s) " "$message" "$default" >&2
  local response
  read -r response
  if test -z "$response"
  then
    response="$default"
  fi
  printf "%s" "$response"
}

# Print a message and get confirmation.
prompt_confirm() {
  local message="${1:-Text}"
  local default="${2:-n}"
  local selection
  case "$default" in
    (y|Y|yes|Yes|YES)
      default=y
      selection="Y/n"
      ;;
    (n|N|no|No|NO)
      default=n
      selection="y/N"
      ;;
    (*)
      echo "Invalid default value: $default" >&2
      return 1
  esac
  printf "%s [%s]: " "$message" "$selection" >&2
  local response
  response="$(get_key)"
  if test -z "$response"
  then
    response="$default"
  fi
  echo "$response" >&2
  case "$response" in
    (y|Y)
      return 0
      ;;
    (n|N)
      return 1
      ;;
  esac
}

# Create a file from the standard input if it does not exist.
ensure_file() {
  local file_path="$1"
  if test -f "$file_path"
  then
    echo "File $file_path already exists. Skipping creation." >&2
    return 0
  fi
  echo "Creating file $file_path." >&2
  mkdir -p "$(dirname "$file_path")"
  cat >"$file_path"
}

underline() {
  printf '\033[4m%s\033[0m' "$1"
}

bold() {
  printf '\033[1m%s\033[0m' "$1"
}

enclose_with_brackets() {
  printf '[%s]' "$1"
}

# Emphasize text.
emph() {
  if test -z "$1"
  then
    return
  fi
  if is_windows
  then
    enclose_with_brackets "$(bold "$(underline "$1")")"
  else
    bold "$(underline "$1")"
  fi
}

# Sort version strings.
# Version strings that are composed of three parts are sorted considering the third part as a patch version.
# Long option `--version-sort` is a GNU sort(1) extension.
# shellcheck disable=SC2120
sort_version() {
  sed -E -e '/-/! { s/^([^.]+(\.[^.]+){2})$/\1_/; }' -e 's/-patch/_patch/' | sort -V "$@" | sed -e 's/_$//' -e 's/_patch/-patch/'
}

# Check if the version is greater than the specified version.
version_gt() {
  test "$(printf '%s\n' "$@" | sort_version | head -n 1)" != "$1"
}

version_ge() {
  test "$(printf '%s\n' "$@" | sort_version -r | head -n 1)" = "$1"
}

# Left/Right-Word-Boundary regex is incompatible with BSD sed // re_format(7) https://man.freebsd.org/cgi/man.cgi?query=re_format&sektion=7
lwb='\<'
rwb='\>'
# shellcheck disable=SC2034
if is_bsd
then
  lwb='[[:<:]]'
  rwb='[[:>:]]'
fi

# Print a menu item with emphasis if a character is prefixed with "&".
menu_item() {
  echo "$1" | sed -E \
    -e 's/&&/@ampersand_ff37f3a@/g' \
    -e "s/^([^&]*)(&([^& ]))?(.*)$/\1${is1}\3${is1}\4/" \
  | (
    IFS="$is1" read -r pre char_to_emph post
    if test -n "$char_to_emph"
    then
      printf -- "%s%s%s" "$pre" "$(emph "$char_to_emph")" "$post"
    else
      printf -- "%s" "$pre"
    fi
  ) | sed -E -e 's/@ampersand_ff37f3a@/\&/g'
  echo
}

# Print a menu
menu() {
  local arg
  for arg in "$@"
  do
    printf -- "- "
    menu_item "$arg"
  done
}

# Get the space-separated nth (1-based) field.
field() {
  # shellcheck disable=SC2046
  printf "%s\n" $(cat) | head -n "$1" | tail -n 1
}

# tac(1) for macOS environment.
if ! command -v tac >/dev/null 2>&1
then
  tac() {
    tail -r
  }
fi

# Check if a directory is empty.
is_dir_empty() {
  if ! test -d "$1"
  then
    return 1
  fi
  if ! test -e "$1"/* 2>/dev/null
  then
    return 0
  fi
  return 1
}

# [<file>] Read the file and print substituting environment variables. Unlike envsubst(1), this tries to expand undefined environment variables and fails for that.
env_subst() {
  local template_file="$1"
  eval "cat <<EOF
$(cat "$template_file")
EOF"
}

# [regex replacement ...] Substitute text that matches regex patterns in stdin input. Takes pairs of regex/replacement arguments and applies them via sed(1).
resubst() {
  local step=2
  local i=0 n=$(($# / step))
  while test "$i" -lt "$n"
  do
    set -- "$@" -e "s${us}$1${us}$2${us}g"
    shift $step
    i=$((i + 1))
  done
  sed "$@"
}

#endregion
