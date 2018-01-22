package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// Graph represents a dependency graph.
type Graph struct {
	Nodes   []*Node
	Sources []*Node
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

func splitManifestTarget(str string) (manifestFile, targetName string) {
	s := strings.Split(str, ":")
	if len(s) == 0 {
		return manifestFile, targetName
	}
	manifestFile = strings.Join(s[:len(s)-1], ":")
	targetName = s[len(s)-1]
	return manifestFile, targetName
}

func parseGraph(manifestFile string) (*Graph, error) {
	if len(manifestFile) == 0 {
		log.Fatalln("error: Please specify a manifest file.")
	}

	manifestMap := map[string]*Manifest{}
	targetNames := []string{}
	nodes := map[string]*Node{}
	targets := map[string]Target{}

	manifestFiles := []string{manifestFile}
	for len(manifestFiles) > 0 {
		manifestFile, manifestFiles = manifestFiles[0], manifestFiles[1:]

		normalized, err := normalizeConfigFile(manifestFile)
		if err != nil {
			return nil, err
		}

		if _, ok := manifestMap[normalized]; ok {
			// NOTE: Skip text that are already read.
			continue
		}

		if _, err := os.Stat(manifestFile); os.IsNotExist(err) {
			log.Fatalf("error: %s does not exist.", manifestFile)
		}

		var manifest Manifest
		if _, err := toml.DecodeFile(manifestFile, &manifest); err != nil {
			return nil, err
		}

		baseDir := filepath.Dir(manifestFile)
		requiredManifests := []string{}

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

			for _, conf := range target.Configs {
				configFile, _ := splitManifestTarget(conf)
				if len(configFile) > 0 {
					requiredManifests = append(requiredManifests, configFile)
				}
			}
			for _, conf := range target.Dependencies {
				configFile, _ := splitManifestTarget(conf)
				if len(configFile) > 0 {
					requiredManifests = append(requiredManifests, configFile)
				}
			}

			node := &Node{
				Name:            target.Name,
				Type:            outputType,
				Headers:         normalizePathList(baseDir, target.Headers),
				Sources:         normalizePathList(baseDir, target.Sources),
				IncludeDirs:     normalizePathList(baseDir, target.IncludeDirs),
				LibDirs:         normalizePathList(baseDir, target.LibDirs),
				Defines:         target.Defines,
				CompilerFlags:   target.CompilerFlags,
				CompilerFlagsC:  target.CompilerFlagsC,
				CompilerFlagsCC: target.CompilerFlagsCC,
				LinkerFlags:     target.LinkerFlags,
				MSBuildSettings: target.MSBuildSettings,
				MSBuildProject:  target.MSBuildProject,
				Templates:       target.Templates,
			}

			node.Tagged = map[string]*Node{}
			for tag, tagged := range target.Tagged {
				node.Tagged[tag] = &Node{
					Headers:         normalizePathList(baseDir, tagged.Headers),
					Sources:         normalizePathList(baseDir, tagged.Sources),
					IncludeDirs:     normalizePathList(baseDir, tagged.IncludeDirs),
					LibDirs:         normalizePathList(baseDir, tagged.LibDirs),
					Defines:         tagged.Defines,
					CompilerFlags:   tagged.CompilerFlags,
					CompilerFlagsC:  tagged.CompilerFlagsC,
					CompilerFlagsCC: tagged.CompilerFlagsCC,
					LinkerFlags:     tagged.LinkerFlags,
					MSBuildSettings: tagged.MSBuildSettings,
					Templates:       tagged.Templates,
				}
			}

			targetNames = append(targetNames, target.Name)
			nodes[target.Name] = node
			targets[target.Name] = target
		}

		manifestMap[normalized] = &manifest

		requiredManifests = normalizePathList(baseDir, requiredManifests)
		manifestFiles = append(requiredManifests, manifestFiles...)
	}

	depNodes := map[string]*Node{}

	for _, name := range targetNames {
		node := nodes[name]
		target := targets[name]
		node.Dependencies = make([]*Node, 0, len(target.Dependencies))
		for _, v := range target.Dependencies {
			_, depName := splitManifestTarget(v)
			dep := nodes[depName]
			node.Dependencies = append(node.Dependencies, dep)
			depNodes[dep.Name] = dep
		}

		node.Configs = make([]*Node, 0, len(target.Configs))
		for _, v := range target.Configs {
			_, configName := splitManifestTarget(v)
			config := nodes[configName]
			node.Configs = append(node.Configs, config)
		}
	}

	sourceNodes := []*Node{}

	for name, node := range nodes {
		if _, ok := depNodes[name]; ok {
			// NOTE: This node is not source.
			continue
		}
		sourceNodes = append(sourceNodes, node)
	}

	orderedNodes := make([]*Node, 0, len(nodes))
	for _, name := range targetNames {
		node := nodes[name]
		orderedNodes = append(orderedNodes, node)
	}

	graph := &Graph{
		Nodes:   orderedNodes,
		Sources: sourceNodes,
	}
	return graph, nil
}
