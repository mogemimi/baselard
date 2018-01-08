package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Tagged struct {
	Headers      []string `toml:"headers"`
	Sources      []string `toml:"sources"`
	IncludeDirs  []string `toml:"include_dirs"`
	Defines      []string `toml:"defines"`
	Dependencies []string `toml:"deps"`
}

type Executable struct {
	Name         string            `toml:"name"`
	Headers      []string          `toml:"headers"`
	Sources      []string          `toml:"sources"`
	IncludeDirs  []string          `toml:"include_dirs"`
	Defines      []string          `toml:"defines"`
	Dependencies []string          `toml:"deps"`
	Tagged       map[string]Tagged `toml:"tagged"`
}

type Manifest struct {
	Imports         []string     `toml:"import"`
	StaticLibraries []Executable `toml:"static_library"`
	Executables     []Executable `toml:"executable"`
}

type Edge struct {
	Name         string
	Type         string
	Headers      []string
	Sources      []string
	IncludeDirs  []string
	Defines      []string
	Dependencies []*Edge
}

type Environment struct {
	OutDir string
}

func (e *Executable) NormalizePaths(base string) {
	for i := range e.Sources {
		src := &e.Sources[i]
		*src = filepath.Clean(filepath.Join(base, *src))
	}
	for i := range e.Headers {
		src := &e.Headers[i]
		*src = filepath.Clean(filepath.Join(base, *src))
	}
	for i := range e.IncludeDirs {
		src := &e.IncludeDirs[i]
		*src = filepath.Clean(filepath.Join(base, *src))
	}
}

func (conf *Manifest) NormalizePaths(configFile string) {
	base := filepath.Dir(configFile)

	for i := range conf.Imports {
		src := &conf.Imports[i]
		*src = filepath.Clean(filepath.Join(base, *src))
	}
	for i := range conf.Executables {
		executable := &conf.Executables[i]
		executable.NormalizePaths(base)
	}
	for i := range conf.StaticLibraries {
		executable := &conf.StaticLibraries[i]
		executable.NormalizePaths(base)
	}
}

func normalizeConfigFile(filename string) (string, error) {
	if !filepath.IsAbs(filename) {
		abs, err := filepath.Abs(filename)
		if err != nil {
			return filename, err
		}
		filename = abs
	}
	filename = filepath.Clean(filename)
	return filename, nil
}

func main() {
	var manifestFile string

	flag.StringVar(&manifestFile, "f", "", "specify a manifest file.")
	flag.Parse()

	if len(manifestFile) == 0 {
		log.Fatalln("error: Please specify a manifest file.")
	}

	manifestMap := map[string]*Manifest{}
	edgeMap := map[string]*Edge{}
	executableMap := map[string]Executable{}

	manifestFiles := []string{manifestFile}
	for len(manifestFiles) > 0 {
		manifestFile, manifestFiles = manifestFiles[0], manifestFiles[1:]

		normalized, err := normalizeConfigFile(manifestFile)
		if err != nil {
			fmt.Println(err)
			return
		}

		if manifestMap[normalized] != nil {
			// NOTE: Skip text that are already read.
			continue
		}

		if _, err := os.Stat(manifestFile); os.IsNotExist(err) {
			log.Fatalf("error: %s does not exist.", manifestFile)
		}

		var manifest Manifest
		if _, err := toml.DecodeFile(manifestFile, &manifest); err != nil {
			fmt.Println(err)
			return
		}
		manifest.NormalizePaths(manifestFile)

		for _, p := range manifest.Executables {
			edge := &Edge{
				Name:        p.Name,
				Type:        "executable",
				Headers:     p.Headers,
				Sources:     p.Sources,
				IncludeDirs: p.IncludeDirs,
				Defines:     p.Defines,
			}
			edgeMap[p.Name] = edge
			executableMap[p.Name] = p
		}

		for _, p := range manifest.StaticLibraries {
			edge := &Edge{
				Name:        p.Name,
				Type:        "static_library",
				Headers:     p.Headers,
				Sources:     p.Sources,
				IncludeDirs: p.IncludeDirs,
				Defines:     p.Defines,
			}
			edgeMap[p.Name] = edge
			executableMap[p.Name] = p
		}

		manifestMap[normalized] = &manifest
		manifestFiles = append(manifest.Imports, manifestFiles...)
	}

	depEdges := map[string]*Edge{}

	for name, edge := range edgeMap {
		executable := executableMap[name]
		deps := make([]*Edge, 0, len(executable.Dependencies))
		for _, v := range executable.Dependencies {
			dep := edgeMap[v]
			deps = append(deps, dep)
			depEdges[dep.Name] = dep
		}
		edge.Dependencies = deps
	}

	sourceEdges := []*Edge{}

	for name, edge := range edgeMap {
		if depEdges[name] != nil {
			// NOTE: This edge is not source.
			continue
		}
		sourceEdges = append(sourceEdges, edge)
	}

	env := &Environment{
		OutDir: "out",
	}

	generator := &NinjaGenerator{}
	generator.Generate(env, edgeMap)

	ninjaFile := "build.ninja"

	err := generator.WriteFile(ninjaFile)
	if err != nil {
		log.Fatalln("error:", err)
	}

	fmt.Println("Generate", ninjaFile)
}
