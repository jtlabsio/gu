# Go Version Updater

## Download

### One-line install

```bash
curl -fsSL https://raw.githubusercontent.com/jtlabsio/gu/main/INSTALL.sh | bash
```

The installer grabs the latest GitHub release for your OS/architecture, extracts the `gu` binary, and places it into `~/.local/bin`, `~/.bin`, `~/bin`, or `%USERPROFILE%/AppData/Local/Programs/gu/bin` (Windows) (whichever exists and is writable). Pass `GU_INSTALL_DIR=/custom/path` to override the destination or `GU_VERSION=1.1.2` to pin a specific release (`v` prefix optional). The script requires `curl` (or `wget`) plus `tar`/`unzip`. On Windows, run the script through Git Bash or WSL.

### Manual download

Please visit <https://github.com/jtlabsio/gu/releases> to see all available releases...

```bash
# download appropriate release
mkdir -p gu
tar -xvzf gu-v1.2.0-linux-amd64.tar.gz -C ./gu
cd gu
./gu
```

## Usage

![](https://github.com/jtlabsio/gu/blob/main/demo.gif)

```bash
# view latest installable versions
# including featured and unstable
gu -l

# view all installable versions 
# (including archived)
gu -la

# install a specific version
gu 1.19.4

# show the help message
gu
```

## Compatibility

This application is designed to work on MacOS, Linux and Windows.

## Notes

Still a work in progress. Please report any defects using issues... would love to take on additional contributors.
