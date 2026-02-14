#!/usr/bin/env bash
set -euo pipefail

Color_Off=''
Red=''
Green=''
Dim=''
Bold_White=''
Bold_Green=''

if [[ -t 1 ]]; then
  Color_Off='\033[0m'
  Red='\033[0;31m'
  Green='\033[0;32m'
  Dim='\033[0;2m'
  Bold_Green='\033[1;32m'
  Bold_White='\033[1m'
fi

error() {
  echo -e "${Red}error${Color_Off}:" "$@" >&2
  exit 1
}

info() {
  echo -e "${Dim}$@${Color_Off}"
}

info_bold() {
  echo -e "${Bold_White}$@${Color_Off}"
}

success() {
  echo -e "${Green}$@${Color_Off}"
}

platform=$(uname -ms)

case $platform in
'Darwin x86_64')
  target=darwin-amd64
  ;;
'Darwin arm64')
  target=darwin-arm64
  ;;
'Linux x86_64' | *)
  target=linux-amd64
  ;;
esac

if [[ ${target} = darwin-amd64 ]]; then
  if [[ $(sysctl -n sysctl.proc_translated 2>/dev/null) = 1 ]]; then
    target=darwin-arm64
    info "Your shell is running in Rosetta 2. Downloading rime for $target instead."
  fi
fi

GITHUB=${GITHUB-"https://github.com"}
github_repo="$GITHUB/rimelabs/rime-cli"

if [[ $# -gt 1 ]]; then
  error "Too many arguments. Pass a version tag as the only argument (e.g. \"v1.0.0\"), or omit to install latest."
fi

if [[ $# = 0 ]]; then
  rime_uri=$github_repo/releases/latest/download/rime-$target.tar.gz
else
  rime_uri=$github_repo/releases/download/$1/rime-$target.tar.gz
fi

install_env=RIME_INSTALL
bin_env=\$$install_env/bin

install_dir=${!install_env:-$HOME/.rime}
bin_dir=$install_dir/bin
exe=$bin_dir/rime

if [[ ! -d $bin_dir ]]; then
  mkdir -p "$bin_dir" ||
    error "Failed to create install directory \"$bin_dir\""
fi

curl --fail --location --progress-bar --output "$exe.tar.gz" "$rime_uri" ||
  error "Failed to download rime from \"$rime_uri\""

tar -xzf "$exe.tar.gz" -C "$bin_dir" ||
  error "Failed to extract rime"

chmod +x "$exe" ||
  error "Failed to set permissions on rime executable"

rm -f "$exe.tar.gz"

case $platform in
'Darwin'*)
  xattr -d com.apple.quarantine "$exe" 2>/dev/null || true
  ;;
esac

tildify() {
  if [[ $1 = $HOME/* ]]; then
    local replacement=\~/
    echo "${1/$HOME\//$replacement}"
  else
    echo "$1"
  fi
}

success "rime was installed successfully to $Bold_Green$(tildify "$exe")"

if command -v rime >/dev/null; then
  echo "Run 'rime --help' to get started"
  exit
fi

refresh_command=''
tilde_bin_dir=$(tildify "$bin_dir")
quoted_install_dir=\"${install_dir//\"/\\\"}\"

if [[ $quoted_install_dir = \"$HOME/* ]]; then
  quoted_install_dir=${quoted_install_dir/$HOME\//\$HOME/}
fi

echo

case $(basename "$SHELL") in
fish)
  commands=(
    "set --export $install_env $quoted_install_dir"
    "set --export PATH $bin_env \$PATH"
  )

  fish_config=$HOME/.config/fish/config.fish
  tilde_fish_config=$(tildify "$fish_config")

  if [[ -w $fish_config ]]; then
    {
      echo -e '\n# rime'
      for command in "${commands[@]}"; do
        echo "$command"
      done
    } >>"$fish_config"

    info "Added \"$tilde_bin_dir\" to \$PATH in \"$tilde_fish_config\""
    refresh_command="source $tilde_fish_config"
  else
    echo "Manually add the directory to $tilde_fish_config (or similar):"
    for command in "${commands[@]}"; do
      info_bold "  $command"
    done
  fi
  ;;
zsh)
  commands=(
    "export $install_env=$quoted_install_dir"
    "export PATH=\"$bin_env:\$PATH\""
  )

  zsh_config=$HOME/.zshrc
  tilde_zsh_config=$(tildify "$zsh_config")

  if [[ -w $zsh_config ]]; then
    {
      echo -e '\n# rime'
      for command in "${commands[@]}"; do
        echo "$command"
      done
    } >>"$zsh_config"

    info "Added \"$tilde_bin_dir\" to \$PATH in \"$tilde_zsh_config\""
    refresh_command="exec $SHELL"
  else
    echo "Manually add the directory to $tilde_zsh_config (or similar):"
    for command in "${commands[@]}"; do
      info_bold "  $command"
    done
  fi
  ;;
bash)
  commands=(
    "export $install_env=$quoted_install_dir"
    "export PATH=\"$bin_env:\$PATH\""
  )

  bash_configs=(
    "$HOME/.bash_profile"
    "$HOME/.bashrc"
  )

  if [[ ${XDG_CONFIG_HOME:-} ]]; then
    bash_configs+=(
      "$XDG_CONFIG_HOME/.bash_profile"
      "$XDG_CONFIG_HOME/.bashrc"
      "$XDG_CONFIG_HOME/bash_profile"
      "$XDG_CONFIG_HOME/bashrc"
    )
  fi

  set_manually=true
  for bash_config in "${bash_configs[@]}"; do
    tilde_bash_config=$(tildify "$bash_config")

    if [[ -w $bash_config ]]; then
      {
        echo -e '\n# rime'
        for command in "${commands[@]}"; do
          echo "$command"
        done
      } >>"$bash_config"

      info "Added \"$tilde_bin_dir\" to \$PATH in \"$tilde_bash_config\""
      refresh_command="source $bash_config"
      set_manually=false
      break
    fi
  done

  if [[ $set_manually = true ]]; then
    echo "Manually add the directory to ~/.bashrc (or similar):"
    for command in "${commands[@]}"; do
      info_bold "  $command"
    done
  fi
  ;;
*)
  echo 'Manually add the directory to ~/.bashrc (or similar):'
  info_bold "  export $install_env=$quoted_install_dir"
  info_bold "  export PATH=\"$bin_env:\$PATH\""
  ;;
esac

echo
info "To get started, run:"
echo

if [[ $refresh_command ]]; then
  info_bold "  $refresh_command"
fi

info_bold "  rime --help"
