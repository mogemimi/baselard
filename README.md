# Baselard

A experimental build system for C++ projects. Still work in progress.

```
    Inputs
+-------------+
|    tags     +--------------------------+
+-------------+                          |
                     Graphs              |           Generators
+-------------+  +-------------+  +------v------+  +-------------+
|  build.toml +-->    edges    +-->    eval     +-->    ninja    |
+-------------+  +-----+-------+  +-------------+  +-------------+
                       |
                       |          +-------------+  +-------------+
                       +----------> MSBuildConf +-->   MSBuild   |
                       |          +-------------+  +-------------+
                       |
                       |          +-------------+  +-------------+
                       +---------->  XcodeConf  +-->    Xcode    |
                                  +-------------+  +-------------+

```

## Build

```shell
$ go get -u github.com/spf13/cobra/cobra
$ go get -u github.com/BurntSushi/toml
$ go build
```

Requirements:

- Go >= 1.9

## Usage

```shell
$ cd path/to/baselard
$ go build
$ ./baselard ninja -f examples/app/build.toml

# Build for macOS
$ ./baselard ninja -f examples/app/build.toml -t mac -t apple
```

## TODO

#### Format

- [ ] depends
- [ ] import/include
- [ ] variables

#### Generator

- [x] Ninja
- [ ] MSBuild and Visual Studio
- [ ] CMake
- [ ] Xcode
- [ ] qmake
- [ ] Make
