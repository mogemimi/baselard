## Build

```shell
cd baselard

# Pull the repo
git clone git@github.com:madler/zlib.git examples/zlib/zlib

# Build with baselard
baselard ninja -i examples/zlib/build.toml
ninja
```
