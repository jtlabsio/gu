# Go Version Updater

## Download

Please visit <https://github.com/jtlabsio/gu/releases> to see all available releases...

```bash
# download appropriate release
mkdir -p gu
tar -xvzf gu-v1.0.0-linux-amd64.tar.gz -C ./gu
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

This was developed on Linux, but it should theoretically work on Windows and Mac OS as well (though this hasn't been tested as of this release). 

## Notes

Still a work in progress. Please report any defects using issues... would love to take on additional contributors.