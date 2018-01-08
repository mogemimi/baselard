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

# For generally usage
$ ./baselard ninja -f examples/app/build.toml

# Building C++ projects for macOS with Ninja
$ ./baselard ninja -f examples/app/build.toml -t mac -t apple
$ ninja

# Generating Visual Studio projects (WIP)
$ ./baselard msbuild -f examples/app/build.toml -o out

# Generating Xcode projects (WIP)
$ ./baselard xcode -f examples/app/build.toml -o out
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
