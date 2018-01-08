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
	Headers       []string `toml:"headers"`
	Sources       []string `toml:"sources"`
	IncludeDirs   []string `toml:"include_dirs"`
	LibDirs       []string `toml:"lib_dirs"`
	Defines       []string `toml:"defines"`
	Dependencies  []string `toml:"deps"`
	CompilerFlags []string `toml:"cflags"`
	LinkerFlags   []string `toml:"ldflags"`
}

type Target struct {
	Name          string            `toml:"name"`
	Type          string            `toml:"type"`
	Headers       []string          `toml:"headers"`
	Sources       []string          `toml:"sources"`
	IncludeDirs   []string          `toml:"include_dirs"`
	LibDirs       []string          `toml:"lib_dirs"`
	Defines       []string          `toml:"defines"`
	CompilerFlags []string          `toml:"cflags"`
	LinkerFlags   []string          `toml:"ldflags"`
	Dependencies  []string          `toml:"deps"`
	Configs       []string          `toml:"configs"`
	Tagged        map[string]Tagged `toml:"tagged"`
}

type Manifest struct {
	Imports []string `toml:"import"`
	Targets []Target `toml:"targets"`
}

type Environment struct {
	OutDir string
}

func (e *Target) NormalizePaths(base string) {
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
	for i := range e.LibDirs {
		src := &e.LibDirs[i]
		*src = filepath.Clean(filepath.Join(base, *src))
	}
}

func (conf *Manifest) NormalizePaths(configFile string) {
	base := filepath.Dir(configFile)

	for i := range conf.Imports {
		src := &conf.Imports[i]
		*src = filepath.Clean(filepath.Join(base, *src))
	}
	for i := range conf.Targets {
		target := &conf.Targets[i]
		target.NormalizePaths(base)
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
	edges := map[string]*Edge{}
	targets := map[string]Target{}

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

		for _, target := range manifest.Targets {
			outputType := func() OutputType {
				switch target.Type {
				case "executable":
					return OutputTypeExecutable
				case "static_library":
					return OutputTypeStaticLibrary
				}
				if len(target.Type) > 0 {
					fmt.Println("warning: Unknown type", target.Type)
				}
				return OutputTypeUnknown
			}()

			edge := &Edge{
				Name:          target.Name,
				Type:          outputType,
				Headers:       target.Headers,
				Sources:       target.Sources,
				IncludeDirs:   target.IncludeDirs,
				LibDirs:       target.LibDirs,
				Defines:       target.Defines,
				CompilerFlags: target.CompilerFlags,
				LinkerFlags:   target.LinkerFlags,
			}

			edges[target.Name] = edge
			targets[target.Name] = target
		}

		manifestMap[normalized] = &manifest
		manifestFiles = append(manifest.Imports, manifestFiles...)
	}

	depEdges := map[string]*Edge{}

	for name, edge := range edges {
		target := targets[name]
		edge.Dependencies = make([]*Edge, 0, len(target.Dependencies))
		for _, v := range target.Dependencies {
			dep := edges[v]
			edge.Dependencies = append(edge.Dependencies, dep)
			depEdges[dep.Name] = dep
		}

		edge.Configs = make([]*Edge, 0, len(target.Configs))
		for _, v := range target.Configs {
			config := edges[v]
			edge.Configs = append(edge.Configs, config)
		}
	}

	sourceEdges := []*Edge{}

	for name, edge := range edges {
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
	generator.Generate(env, edges)

	ninjaFile := "build.ninja"

	err := generator.WriteFile(ninjaFile)
	if err != nil {
		log.Fatalln("error:", err)
	}

	fmt.Println("Generate", ninjaFile)
}
