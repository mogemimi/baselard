## Build

```shell
cd baselard

# Pull the repo
git clone git@github.com:madler/zlib.git examples/zlib/zlib
git clone git@github.com:glennrp/libpng.git examples/libpng/libpng

# Build with baselard
baselard ninja -i examples/libpng/build.toml
ninja
```
