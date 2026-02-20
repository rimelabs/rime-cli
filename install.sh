#!/bin/sh
set -eu

# ── Colors ────────────────────────────────────────────────────────────────────

Color_Off=''
Red=''
Green=''
Dim=''
Bold_White=''
Bold_Green=''

if [ -t 1 ]; then
  Color_Off='\033[0m'
  Red='\033[0;31m'
  Green='\033[0;32m'
  Dim='\033[0;2m'
  Bold_Green='\033[1;32m'
  Bold_White='\033[1m'
fi

# ── Helpers ───────────────────────────────────────────────────────────────────

error() {
  printf '%b\n' "${Red}error${Color_Off}: $*" >&2
  exit 1
}

info() {
  printf '%b\n' "${Dim}$*${Color_Off}"
}

info_bold() {
  printf '%b\n' "${Bold_White}$*${Color_Off}"
}

success() {
  printf '%b\n' "${Green}$*${Color_Off}"
}

divider() {
  printf '%b\n' "${Dim}──────────────────────────────────────${Color_Off}"
}

tildify() {
  case "$1" in
  "$HOME"/*)
    printf '%s\n' "~/${1#"$HOME"/}"
    ;;
  *)
    printf '%s\n' "$1"
    ;;
  esac
}

# ── Argument parsing ──────────────────────────────────────────────────────────

YES=false
VERSION=''

for arg in "$@"; do
  case "$arg" in
  -y | --yes)
    YES=true
    ;;
  -h | --help)
    echo "Usage: install.sh [options] [version]"
    echo ""
    echo "Options:"
    echo "  -y, --yes    Skip interactive prompts (useful for CI)"
    echo "  -h, --help   Show this help"
    echo ""
    echo "Arguments:"
    echo "  version      Version to install (e.g. v1.0.0). Defaults to latest."
    exit 0
    ;;
  -*)
    error "Unknown option: $arg. Run with --help for usage."
    ;;
  *)
    if [ -n "$VERSION" ]; then
      error "Too many arguments. Pass a version tag as the only argument (e.g. \"v1.0.0\"), or omit to install latest."
    fi
    VERSION="$arg"
    ;;
  esac
done

# ── Platform detection ────────────────────────────────────────────────────────

platform="$(uname -s) $(uname -m)"

case "$platform" in
'Darwin x86_64')
  target=darwin-amd64
  ;;
'Darwin arm64')
  target=darwin-arm64
  ;;
'Linux x86_64')
  target=linux-amd64
  ;;
*)
  error "Unsupported platform: $platform. Rime CLI supports macOS (x86_64, arm64) and Linux (x86_64)."
  ;;
esac

if [ "$target" = "darwin-amd64" ]; then
  translated="$(sysctl -n sysctl.proc_translated 2>/dev/null || true)"
  if [ "$translated" = "1" ]; then
    target=darwin-arm64
    info "Your shell is running in Rosetta 2. Downloading rime for $target instead."
  fi
fi

# ── Paths & URLs ──────────────────────────────────────────────────────────────

github_repo='https://github.com/rimelabs/rime-cli'

if [ -z "$VERSION" ]; then
  rime_uri=$github_repo/releases/latest/download/rime-$target.tar.gz
else
  rime_uri=$github_repo/releases/download/$VERSION/rime-$target.tar.gz
fi

install_dir=${RIME_INSTALL:-$HOME/.rime}
bin_dir=$install_dir/bin
exe=$bin_dir/rime
tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/rime-install.XXXXXX")" ||
  error "Failed to create temporary directory"
tarball=$tmp_dir/rime.tar.gz

# ── Cleanup trap ──────────────────────────────────────────────────────────────

cleanup() {
  rm -f "$tarball"
  rm -rf "$tmp_dir"
}
trap cleanup EXIT

# ── Welcome ───────────────────────────────────────────────────────────────────

echo
printf '%b\n' "${Bold_White}Installing Rime CLI...${Color_Off}"
echo

# ── Snapshot existing state ───────────────────────────────────────────────────

existing_version=''
already_in_path=false
if command -v rime >/dev/null 2>&1; then
  already_in_path=true
  existing_version=$(rime --version 2>/dev/null | head -1 || true)
fi

# ── Download / install ────────────────────────────────────────────────────────

if [ ! -d "$bin_dir" ]; then
  mkdir -p "$bin_dir" ||
    error "Failed to create install directory \"$bin_dir\""
fi

curl --fail --location --output "$tarball" "$rime_uri" ||
  error "Failed to download rime from \"$rime_uri\""

tar_entries="$(tar -tzf "$tarball")" ||
  error "Failed to inspect downloaded archive"

has_binary=false
while IFS= read -r entry; do
  [ -z "$entry" ] && continue
  case "$entry" in
  /* | ../* | */../* | ..)
    error "Archive contains unsafe path: $entry"
    ;;
  esac
  case "$entry" in
  rime | ./rime | */rime)
    has_binary=true
    ;;
  esac
done <<EOF
$tar_entries
EOF

[ "$has_binary" = "true" ] || error "Archive does not contain rime binary"

tar -xzf "$tarball" -C "$tmp_dir" ||
  error "Failed to extract rime archive"

src_exe=''
if [ -f "$tmp_dir/rime" ]; then
  src_exe="$tmp_dir/rime"
elif [ -f "$tmp_dir/bin/rime" ]; then
  src_exe="$tmp_dir/bin/rime"
else
  src_exe=$(printf '%s\n' "$tar_entries" | sed -n 's|^\./||; /\/rime$/p; /^rime$/p' | head -1)
  [ -n "$src_exe" ] || error "Could not locate rime binary in archive"
  src_exe="$tmp_dir/$src_exe"
fi

[ -f "$src_exe" ] || error "Extracted rime binary is missing"
cp "$src_exe" "$exe" ||
  error "Failed to install rime binary"

chmod +x "$exe" ||
  error "Failed to set permissions on rime executable"

case "$platform" in
'Darwin'*)
  xattr -d com.apple.quarantine "$exe" 2>/dev/null || true
  ;;
esac

# ── Write env files ───────────────────────────────────────────────────────────
# These are sourced by a single line added to the user's shell config,
# making uninstall clean (one line to remove)

{
  echo '# rime'
  echo "export RIME_INSTALL=\"$install_dir\""
  echo 'export PATH="$RIME_INSTALL/bin:$PATH"'
} >"$install_dir/env.sh"

{
  echo '# rime'
  echo "set --export RIME_INSTALL \"$install_dir\""
  echo "set --export PATH \"$install_dir/bin\" \$PATH"
} >"$install_dir/env.fish"

# ── Report result ─────────────────────────────────────────────────────────────

new_version=$("$exe" --version 2>/dev/null | head -1 || true)

if [ -n "$existing_version" ] && [ -n "$new_version" ] && [ "$existing_version" != "$new_version" ]; then
  success "✓ Upgraded rime  $existing_version → $new_version"
elif [ -n "$new_version" ]; then
  success "✓ Installed $new_version"
else
  success "✓ Rime CLI installed to $Bold_Green$(tildify "$exe")${Color_Off}"
fi

echo

# ── Interactive prompt helper ─────────────────────────────────────────────────

ask_yes_no() {
  prompt="$1"

  if [ "$YES" = "true" ]; then
    printf '%b\n' "$prompt ${Dim}[Y/n] Y${Color_Off}"
    return 0
  fi

  answer=''
  if [ -t 0 ]; then
    printf '%b' "$prompt ${Dim}[Y/n]${Color_Off} "
    IFS= read -r answer || true
  elif [ -e /dev/tty ]; then
    # Running via curl | sh — stdin is the pipe, but we can still prompt via tty
    printf '%b' "$prompt ${Dim}[Y/n]${Color_Off} " >/dev/tty
    IFS= read -r answer </dev/tty || true
  else
    # No interactive input available; default to no to avoid mutating shell config.
    printf '%b\n' "$prompt ${Dim}[Y/n] N${Color_Off}"
    return 1
  fi

  answer=${answer:-y}
  case "$answer" in
  [Yy]) return 0 ;;
  *) return 1 ;;
  esac
}

# ── PATH setup ────────────────────────────────────────────────────────────────

refresh_command=''

if [ "$already_in_path" = "false" ]; then
  divider
  printf '%b\n' "  ${Bold_White}PATH setup${Color_Off}"
  divider
  echo

  tilde_bin_dir=$(tildify "$bin_dir")

  case "$(basename "$SHELL")" in
  fish)
    fish_config=$HOME/.config/fish/config.fish
    tilde_fish_config=$(tildify "$fish_config")
    source_line="source \"$install_dir/env.fish\""

    if ask_yes_no "  Add $tilde_bin_dir to your PATH? (modifies $tilde_fish_config)"; then
      if [ -w "$fish_config" ]; then
        if ! grep -qF "$source_line" "$fish_config" 2>/dev/null; then
          {
            echo
            echo '# rime'
            echo "$source_line"
          } >>"$fish_config"
        fi
        echo
        success "  ✓ Updated $tilde_fish_config"
        refresh_command="source $tilde_fish_config"
      else
        echo
        info "  Could not write to $tilde_fish_config. Add manually:"
        info_bold "    $source_line"
      fi
    else
      echo
      info "  Skipped. To add manually:"
      info_bold "    $source_line"
    fi
    ;;

  zsh)
    zsh_config=$HOME/.zshrc
    tilde_zsh_config=$(tildify "$zsh_config")
    source_line=". \"$install_dir/env.sh\""

    if ask_yes_no "  Add $tilde_bin_dir to your PATH? (modifies $tilde_zsh_config)"; then
      if [ -w "$zsh_config" ]; then
        if ! grep -qF "$source_line" "$zsh_config" 2>/dev/null; then
          {
            echo
            echo '# rime'
            echo "$source_line"
          } >>"$zsh_config"
        fi
        echo
        success "  ✓ Updated $tilde_zsh_config"
        refresh_command="source $tilde_zsh_config"
      else
        echo
        info "  Could not write to $tilde_zsh_config. Add manually:"
        info_bold "    $source_line"
      fi
    else
      echo
      info "  Skipped. To add manually:"
      info_bold "    $source_line"
    fi
    ;;

  bash)
    bash_config=''
    if [ -w "$HOME/.bash_profile" ]; then
      bash_config="$HOME/.bash_profile"
    elif [ -w "$HOME/.bashrc" ]; then
      bash_config="$HOME/.bashrc"
    elif [ -n "${XDG_CONFIG_HOME:-}" ] && [ -w "$XDG_CONFIG_HOME/.bash_profile" ]; then
      bash_config="$XDG_CONFIG_HOME/.bash_profile"
    elif [ -n "${XDG_CONFIG_HOME:-}" ] && [ -w "$XDG_CONFIG_HOME/.bashrc" ]; then
      bash_config="$XDG_CONFIG_HOME/.bashrc"
    elif [ -n "${XDG_CONFIG_HOME:-}" ] && [ -w "$XDG_CONFIG_HOME/bash_profile" ]; then
      bash_config="$XDG_CONFIG_HOME/bash_profile"
    elif [ -n "${XDG_CONFIG_HOME:-}" ] && [ -w "$XDG_CONFIG_HOME/bashrc" ]; then
      bash_config="$XDG_CONFIG_HOME/bashrc"
    fi

    if [ -n "$bash_config" ]; then
      tilde_bash_config=$(tildify "$bash_config")
    else
      tilde_bash_config=$(tildify "$HOME/.bashrc")
    fi
    source_line=". \"$install_dir/env.sh\""

    if ask_yes_no "  Add $tilde_bin_dir to your PATH? (modifies $tilde_bash_config)"; then
      if [ -n "$bash_config" ]; then
        if ! grep -qF "$source_line" "$bash_config" 2>/dev/null; then
          {
            echo
            echo '# rime'
            echo "$source_line"
          } >>"$bash_config"
        fi
        echo
        success "  ✓ Updated $tilde_bash_config"
        refresh_command="source $tilde_bash_config"
      else
        echo
        info "  Could not find a writable bash config. Add manually to ~/.bashrc:"
        info_bold "    $source_line"
      fi
    else
      echo
      info "  Skipped. To add manually:"
      info_bold "    $source_line"
    fi
    ;;

  *)
    echo "  Add rime to your PATH by adding to ~/.bashrc (or similar):"
    info_bold "    . \"$install_dir/env.sh\""
    ;;
  esac

  echo
fi

# ── Get started ───────────────────────────────────────────────────────────────

divider
printf '%b\n' "  ${Bold_White}Get started${Color_Off}"
divider
echo

if [ -f "$install_dir/rime.toml" ]; then
  info "  You're already logged in. Run:"
  echo
  info_bold "    rime --help"
elif [ -n "$refresh_command" ]; then
  info "  Reload your shell, then log in:"
  echo
  info_bold "    $refresh_command"
  info_bold "    rime login"
else
  info "  Log in to get started:"
  echo
  info_bold "    rime login"
fi

echo
info "  Tip: set up tab completion with ${Bold_White}rime completion --help${Color_Off}"
echo
