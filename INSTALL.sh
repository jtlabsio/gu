#!/usr/bin/env bash
set -euo pipefail

REPO="jtlabsio/gu"
BIN_NAME="gu"

abort() {
  printf 'Error: %s\n' "$*" >&2
  exit 1
}

command_exists() {
  command -v "$1" >/dev/null 2>&1
}

require_cmd() {
  command_exists "$1" || abort "Required command '$1' not found in PATH."
}

detect_download_tool() {
  if command_exists curl; then
    printf 'curl'
  elif command_exists wget; then
    printf 'wget'
  else
    abort "Need either 'curl' or 'wget' to download releases."
  fi
}

fetch_json_field() {
  local json=$1
  local key=$2
  printf '%s' "$json" | grep -m 1 "\"$key\"" | sed -E 's/.*"'$key'": *"([^"]+)".*/\1/'
}

detect_os() {
  local uname_out
  uname_out=$(uname -s | tr '[:upper:]' '[:lower:]')
  case "$uname_out" in
    linux*) printf 'linux' ;;
    darwin*) printf 'darwin' ;;
    msys*|mingw*|cygwin*|nt*) printf 'windows' ;;
    *) abort "Unsupported operating system '$uname_out'." ;;
  esac
}

detect_arch() {
  local uname_m
  uname_m=$(uname -m | tr '[:upper:]' '[:lower:]')
  case "$uname_m" in
    x86_64|amd64) printf 'amd64' ;;
    arm64|aarch64) printf 'arm64' ;;
    i386|i686|x86) printf '386' ;;
    *) abort "Unsupported architecture '$uname_m'." ;;
  esac
}

normalize_tag() {
  local tag=$1
  if [[ "$tag" != v* ]]; then
    printf 'v%s' "$tag"
  else
    printf '%s' "$tag"
  fi
}

choose_install_dir() {
  local os=$1
  if [[ -n "${GU_INSTALL_DIR:-}" ]]; then
    printf '%s\n' "$GU_INSTALL_DIR"
    return
  fi

  local -a candidates
  if [[ "$os" == "windows" ]]; then
    candidates=("$HOME/AppData/Local/Programs/gu/bin" "$HOME/bin" "$HOME/.local/bin" "$HOME/.bin")
  else
    candidates=("$HOME/.local/bin" "$HOME/.bin" "$HOME/bin")
  fi

  for dir in "${candidates[@]}"; do
    [[ -z "$dir" ]] && continue
    if [[ ! -d "$dir" ]]; then
      mkdir -p "$dir" 2>/dev/null || continue
    fi
    if [[ -w "$dir" ]]; then
      printf '%s\n' "$dir"
      return
    fi
  done

  abort "Could not find a writable install directory. Set GU_INSTALL_DIR to an absolute path."
}

ensure_tools_for_archive() {
  local ext=$1
  case "$ext" in
    tar.gz) require_cmd tar ;;
    zip) require_cmd unzip ;;
    *) abort "Unknown archive format '$ext'." ;;
  esac
}

download_file() {
  local url=$1
  local dest=$2
  local tool=$3

  if [[ "$tool" == "curl" ]]; then
    curl -fsSL "$url" -o "$dest"
  else
    wget -qO "$dest" "$url"
  fi
}

download_latest_release_info() {
  local tool=$1
  local api="https://api.github.com/repos/${REPO}/releases/latest"
  if [[ "$tool" == "curl" ]]; then
    curl -fsSL "$api"
  else
    wget -qO- "$api"
  fi
}

extract_archive() {
  local archive=$1
  local ext=$2
  local dest_dir=$3

  case "$ext" in
    tar.gz) tar -xzf "$archive" -C "$dest_dir" ;;
    zip) unzip -q "$archive" -d "$dest_dir" ;;
  esac
}

main() {
  local os arch archive_ext tag download_tool release_json asset_name download_url tmp_dir install_dir bin_src bin_dest

  download_tool=$(detect_download_tool)
  os=$(detect_os)
  arch=$(detect_arch)

  if [[ "$os" == "windows" && "$arch" == "arm64" ]]; then
    abort "Windows arm64 builds are not available yet."
  fi

  if [[ -n "${GU_VERSION:-}" ]]; then
    tag=$(normalize_tag "$GU_VERSION")
  else
    release_json=$(download_latest_release_info "$download_tool") || abort "Unable to query GitHub releases."
    release_json=$(printf '%s' "$release_json" | tr -d '\r')
    tag=$(fetch_json_field "$release_json" "tag_name")
    [[ -z "$tag" ]] && abort "Could not determine the latest release tag."
    tag=$(normalize_tag "$tag")
  fi

  if [[ "$os" == "windows" ]]; then
    archive_ext="zip"
  else
    archive_ext="tar.gz"
  fi

  asset_name="${BIN_NAME}-${tag}-${os}-${arch}.${archive_ext}"
  download_url="https://github.com/${REPO}/releases/download/${tag}/${asset_name}"

  ensure_tools_for_archive "$archive_ext"

  tmp_dir=$(mktemp -d)
  trap '[[ -n "${tmp_dir:-}" ]] && rm -rf "$tmp_dir"' EXIT

  printf 'Downloading %s...\n' "$asset_name"
  download_file "$download_url" "${tmp_dir}/${asset_name}" "$download_tool" || abort "Download failed. Verify that release ${tag} supports ${os}/${arch}."

  extract_archive "${tmp_dir}/${asset_name}" "$archive_ext" "$tmp_dir"

  if [[ "$os" == "windows" ]]; then
    bin_src="${tmp_dir}/${BIN_NAME}.exe"
    [[ -f "$bin_src" ]] || abort "Unable to find ${BIN_NAME}.exe in archive."
  else
    bin_src="${tmp_dir}/${BIN_NAME}"
    [[ -f "$bin_src" ]] || abort "Unable to find ${BIN_NAME} in archive."
    chmod +x "$bin_src"
  fi

  install_dir=$(choose_install_dir "$os")
  if [[ "$os" == "windows" ]]; then
    bin_dest="${install_dir}/${BIN_NAME}.exe"
  else
    bin_dest="${install_dir}/${BIN_NAME}"
  fi

  mkdir -p "$install_dir"
  install -m 755 "$bin_src" "$bin_dest" 2>/dev/null || {
    cp "$bin_src" "$bin_dest"
    chmod 755 "$bin_dest"
  }

  printf 'Installed %s to %s\n' "$BIN_NAME" "$bin_dest"

  case ":$PATH:" in
    *":$install_dir:"*) ;;
    *) printf 'Note: %s is not on your PATH. Add it or set GU_INSTALL_DIR.\n' "$install_dir" ;;
  esac

  printf 'Run "%s -h" to get started.\n' "$BIN_NAME"
}

main "$@"
