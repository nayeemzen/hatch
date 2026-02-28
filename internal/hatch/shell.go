package hatch

import (
	"fmt"
	"strings"
)

func shellInit(shell string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(shell)) {
	case "zsh", "bash":
		return posixShellInit(), nil
	case "fish":
		return fishShellInit(), nil
	default:
		return "", fmt.Errorf("unsupported shell %q (use zsh, bash, or fish)", shell)
	}
}

func posixShellInit() string {
	return `hatch() {
  local _hatch_cwd_file
  _hatch_cwd_file="$(mktemp "${TMPDIR:-/tmp}/hatch-cwd.XXXXXX")"
  command hatch --cwd-file "$_hatch_cwd_file" "$@"
  local _hatch_status=$?

  if [ $_hatch_status -eq 0 ] && [ -s "$_hatch_cwd_file" ]; then
    local _hatch_target
    _hatch_target="$(cat "$_hatch_cwd_file")"
    if [ -d "$_hatch_target" ]; then
      cd "$_hatch_target" || return $_hatch_status
    fi
  fi

  rm -f "$_hatch_cwd_file"
  return $_hatch_status
}`
}

func fishShellInit() string {
	return `function hatch
  set -l _hatch_cwd_file (mktemp (string join '' (or $TMPDIR /tmp) '/hatch-cwd.XXXXXX'))
  command hatch --cwd-file "$_hatch_cwd_file" $argv
  set -l _hatch_status $status

  if test $_hatch_status -eq 0
    if test -s "$_hatch_cwd_file"
      set -l _hatch_target (cat "$_hatch_cwd_file")
      if test -d "$_hatch_target"
        cd "$_hatch_target"
      end
    end
  end

  rm -f "$_hatch_cwd_file"
  return $_hatch_status
end`
}
