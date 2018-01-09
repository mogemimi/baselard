package main

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/spf13/cobra"
)

type Tagged struct {
	Headers         []string        `toml:"headers"`
	Sources         []string        `toml:"sources"`
	IncludeDirs     []string        `toml:"include_dirs"`
	LibDirs         []string        `toml:"lib_dirs"`
	Defines         []string        `toml:"defines"`
	Dependencies    []string        `toml:"deps"`
	CompilerFlags   []string        `toml:"cflags"`
	LinkerFlags     []string        `toml:"ldflags"`
	MSBuildSettings MSBuildSettings `toml:"msbuild_settings"`
}

type Target struct {
	Name            string            `toml:"name"`
	Type            string            `toml:"type"`
	Headers         []string          `toml:"headers"`
	Sources         []string          `toml:"sources"`
	IncludeDirs     []string          `toml:"include_dirs"`
	LibDirs         []string          `toml:"lib_dirs"`
	Defines         []string          `toml:"defines"`
	CompilerFlags   []string          `toml:"cflags"`
	LinkerFlags     []string          `toml:"ldflags"`
	MSBuildSettings MSBuildSettings   `toml:"msbuild_settings"`
	Dependencies    []string          `toml:"deps"`
	Configs         []string          `toml:"configs"`
	Tagged          map[string]Tagged `toml:"tagged"`
	MSBuildProject  MSBuildProject    `toml:"msbuild_project"`
}

type MSBuildSettings struct {
	ClCompile         map[string]string `toml:"ClCompile"`
	Link              map[string]string `toml:"Link"`
	Lib               map[string]string `toml:"Lib"`
	Globals           map[string]string `toml:"Globals"`
	Configuration     map[string]string `toml:"Configuration"`
	User              map[string]string `toml:"User"`
	General           map[string]string `toml:"General"`
	ExtensionSettings []string          `toml:"ExtensionSettings"`
}

type MSBuildProjectConfiguration struct {
	Platform      string   `toml:"platform"`
	Configuration string   `toml:"configuration"`
	Tags          []string `toml:"tags"`
}

type MSBuildProject struct {
	Configurations []MSBuildProjectConfiguration `toml:"configurations"`
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
	graph, _, err := parseGraph(manifestFile)
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

func generateMSBuild(manifestFile string) {
	graph, generatorSettings, err := parseGraph(manifestFile)
	if err != nil {
		log.Fatalln("error:", err)
	}

	env := &Environment{
		OutDir: "out",
	}

	generator := &MSBuildGenerator{}
	generator.Generate(env, graph, generatorSettings)

	// err = generator.WriteFile()
	// if err != nil {
	// 	log.Fatalln("error:", err)
	// }

	fmt.Println("Generate Visual Studio project")
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
	ninjaCmd.Flags().StringVarP(&outputNinjaFile, "file", "f", "build.ninja", "specify a output ninja file")

	var msbuildCmd = &cobra.Command{
		Use:   "msbuild",
		Short: "Generate Visual Studio projects",
		Long:  `Ganerate Visual Studio solution and project files.`,
		Run: func(cmd *cobra.Command, args []string) {
			generateMSBuild(manifestFile)
		},
	}

	var rootCmd = &cobra.Command{Use: "baselard"}
	rootCmd.PersistentFlags().StringVarP(&manifestFile, "input", "i", "", "specify a manifest file")
	rootCmd.AddCommand(ninjaCmd, msbuildCmd)
	rootCmd.Execute()
}
