package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Executable struct {
	Name         string   `toml:"name"`
	Headers      []string `toml:"headers"`
	Sources      []string `toml:"sources"`
	IncludeDirs  []string `toml:"include_dirs"`
	Defines      []string `toml:"defines"`
	Dependencies []string `toml:"deps"`
}

type Config struct {
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

func (conf *Config) NormalizePaths(configFile string) {
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
	var configFile string

	flag.StringVar(&configFile, "f", "", "specify a config file.")
	flag.Parse()

	if len(configFile) == 0 {
		log.Fatalln("error: Please specify a configuration file.")
	}

	var rootConf Config

	configMap := map[string]*Config{}

	configFiles := []string{configFile}
	for len(configFiles) > 0 {
		configFile, configFiles = configFiles[0], configFiles[1:]

		normalized, err := normalizeConfigFile(configFile)
		if err != nil {
			fmt.Println(err)
			return
		}

		if configMap[normalized] != nil {
			// NOTE: Skip text that are already read.
			continue
		}

		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			log.Fatalf("error: %s does not exist.", configFile)
		}

		var conf Config
		if _, err := toml.DecodeFile(configFile, &conf); err != nil {
			fmt.Println(err)
			return
		}
		conf.NormalizePaths(configFile)

		configMap[normalized] = &conf
		rootConf.Executables = append(rootConf.Executables, conf.Executables...)
		rootConf.StaticLibraries = append(rootConf.StaticLibraries, conf.StaticLibraries...)

		configFiles = append(conf.Imports, configFiles...)
	}

	var edges []*Edge
	for _, p := range rootConf.Executables {
		// TODO: not implemented
		deps := []*Edge{}

		edge := &Edge{
			Name:         p.Name,
			Type:         "executable",
			Headers:      p.Headers,
			Sources:      p.Sources,
			IncludeDirs:  p.IncludeDirs,
			Defines:      p.Defines,
			Dependencies: deps,
		}
		edges = append(edges, edge)
	}

	env := &Environment{
		OutDir: "out",
	}

	generator := &NinjaGenerator{}
	generator.Generate(env, &rootConf)

	ninjaFile := "build.ninja"

	err := generator.WriteFile(ninjaFile)
	if err != nil {
		log.Fatalln("error:", err)
	}

	fmt.Println("Generate", ninjaFile)
}
