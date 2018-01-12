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
$ go get -u github.com/pkg/errors
$ go get -u github.com/satori/go.uuid
$ go build
```

Requirements:

- Go >= 1.9

## Usage

```shell
$ cd path/to/baselard
$ go build

# For generally usage
$ ./baselard ninja -i examples/app/build.toml

# Building C++ projects for macOS with Ninja
$ ./baselard ninja -i examples/app/build.toml -t mac -t apple
$ ninja

# Generating Visual Studio projects
$ ./baselard msbuild -i examples/app/build.toml -g out
$ MSBuild.exe out/out.sln -t:Build -p:Configuration=Release

# Generating Xcode projects (WIP)
$ ./baselard xcode -i examples/app/build.toml -o out
```

## TODO

#### Format

- [ ] depends
- [ ] import/include
- [ ] variables

#### Generator

- [x] Ninja
  - [ ] Switch compilers between gcc and clang
- [x] MSBuild and Visual Studio
  - [x] Project dependencies
  - [x] `*.sln`
  - [x] `*.vcxproj`
  - [x] `*.vcxproj.filters`
    - [ ] Hierarchical Filters
- [ ] CMake
- [ ] Xcode
- [ ] qmake
- [ ] Make
