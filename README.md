# Baselard

A experimental build system for C++ projects. Still work in progress.

## Build

```shell
$ go get github.com/BurntSushi/toml
$ go build
```

Requirements:

- Go >= 1.9

## Usage

```shell
$ cd path/to/baselard
$ go build
$ baselard -f examples/app/build.toml
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
