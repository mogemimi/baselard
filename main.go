package main

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/spf13/cobra"
)

// Environment specifies a execution environment for project generators.
type Environment struct {
	OutDir         string
	ProjectFileDir string
	Tags           []string
}

func generateNinja(manifestFile string, ninjaFile string, tags []string) {
	graph, err := parseGraph(manifestFile)
	if err != nil {
		log.Fatalln("error:", err)
	}

	env := &Environment{
		OutDir:         "out",
		ProjectFileDir: filepath.Dir(ninjaFile),
		Tags:           tags,
	}

	generator := &NinjaGenerator{}
	generator.Generate(env, graph)

	err = generator.WriteFile(ninjaFile)
	if err != nil {
		log.Fatalln("error:", err)
	}

	fmt.Println("Generate", ninjaFile)
}

func generateMSBuild(manifestFile, outputGenDir string) {
	graph, err := parseGraph(manifestFile)
	if err != nil {
		log.Fatalln("error:", err)
	}

	env := &Environment{
		OutDir:         "out",
		ProjectFileDir: outputGenDir,
	}

	generator := &MSBuildGenerator{}
	generator.Generate(env, graph)

	err = generator.WriteFile(env)
	if err != nil {
		log.Fatalln("error:", err)
	}
}

func main() {
	var manifestFile string
	var outputNinjaFile string
	var outputGenDir string
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
			generateMSBuild(manifestFile, outputGenDir)
		},
	}
	msbuildCmd.Flags().StringVarP(&outputGenDir, "gen-dir", "g", "out", "specify a directory for generated project files")

	var rootCmd = &cobra.Command{Use: "baselard"}
	rootCmd.PersistentFlags().StringVarP(&manifestFile, "input", "i", "", "specify a manifest file")
	rootCmd.AddCommand(ninjaCmd, msbuildCmd)
	rootCmd.Execute()
}
