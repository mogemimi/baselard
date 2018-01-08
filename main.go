package main

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/spf13/cobra"
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
	Tags   []string
}

func normalizePathList(base string, paths []string) (result []string) {
	for _, filename := range paths {
		result = append(result, filepath.Clean(filepath.Join(base, filename)))
	}
	return result
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

func generateNinja(manifestFile string, ninjaFile string, tags []string) {
	graph, err := parseGraph(manifestFile)
	if err != nil {
		log.Fatalln("error:", err)
	}

	env := &Environment{
		OutDir: "out",
		Tags:   tags,
	}

	generator := &NinjaGenerator{}
	generator.Generate(env, graph.edges)

	err = generator.WriteFile(ninjaFile)
	if err != nil {
		log.Fatalln("error:", err)
	}

	fmt.Println("Generate", ninjaFile)
}

func main() {
	var manifestFile string
	var outputNinjaFile string
	var tags []string

	var ninjaCmd = &cobra.Command{
		Use:   "ninja",
		Short: "Generate ninja file",
		Long:  `Ganerate ninja file.`,
		Run: func(cmd *cobra.Command, args []string) {
			generateNinja(manifestFile, outputNinjaFile, tags)
		},
	}
	ninjaCmd.Flags().StringArrayVarP(&tags, "tag", "t", tags, "specify tags")
	ninjaCmd.Flags().StringVarP(&outputNinjaFile, "out", "o", "build.ninja", "specify a output ninja file")

	var msbuildCmd = &cobra.Command{
		Use:   "msbuild",
		Short: "Generate Visual Studio projects",
		Long:  `Ganerate Visual Studio solution and project files.`,
		Run: func(cmd *cobra.Command, args []string) {
		},
	}

	var rootCmd = &cobra.Command{Use: "baselard"}
	rootCmd.PersistentFlags().StringVarP(&manifestFile, "file", "f", "", "specify a manifest file")
	rootCmd.AddCommand(ninjaCmd, msbuildCmd)
	rootCmd.Execute()
}
