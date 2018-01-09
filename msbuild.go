package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/pkg/errors"
)

type MSBuildGenerator struct {
}

func (generator *MSBuildGenerator) Generate(env *Environment, graph *Graph, generatorSettings *GeneratorSettings) {

	for _, edge := range graph.edges {
		fmt.Println("edge.Name =", edge.Name)

		project := edge.GetMSBuildProject(env)

		if edge.Name != "app" {
			continue
		}

		for _, config := range project.Configurations {
			fmt.Println("  ", "if Configuration|Platform =", config.Configuration, "|", config.Platform)
			fmt.Println("    ", "config.Tags =", config.Tags)

			projectEnv := &Environment{}
			projectEnv.OutDir = env.OutDir
			projectEnv.Tags = env.Tags
			projectEnv.Tags = append(projectEnv.Tags, config.Tags...)

			msbuild := edge.GetMSBuildSettings(projectEnv)

			fmt.Println("    ", "ClCompile =", msbuild.ClCompile)
			fmt.Println("    ", "Link =", msbuild.Link)
			fmt.Println("    ", "Lib =", msbuild.Lib)
			fmt.Println("    ", "Globals =", msbuild.Globals)
			fmt.Println("    ", "Configuration =", msbuild.Configuration)
			fmt.Println("    ", "User =", msbuild.User)
			fmt.Println("    ", "General =", msbuild.General)
			fmt.Println("    ", "ExtensionSettings =", msbuild.ExtensionSettings)
		}
	}
}

func (gen *MSBuildGenerator) WriteFile(outputPath string) error {
	dir := filepath.Dir(outputPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return errors.Wrapf(err, "Failed to create output directory \"%s\"", dir)
		}
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	writer.Flush()

	return nil
}
