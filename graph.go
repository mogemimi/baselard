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
	edges   map[string]*Node
	sources []*Node
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
	edges := map[string]*Node{}
	targets := map[string]Target{}

	manifestFiles := []string{manifestFile}
	for len(manifestFiles) > 0 {
		manifestFile, manifestFiles = manifestFiles[0], manifestFiles[1:]

		normalized, err := normalizeConfigFile(manifestFile)
		if err != nil {
			return nil, err
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

			edge := &Node{
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
			}

			edge.Tagged = map[string]*Node{}
			for tag, tagged := range target.Tagged {
				edge.Tagged[tag] = &Node{
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
				}
			}

			edges[target.Name] = edge
			targets[target.Name] = target
		}

		manifestMap[normalized] = &manifest

		requiredManifests = normalizePathList(baseDir, requiredManifests)
		manifestFiles = append(requiredManifests, manifestFiles...)
	}

	depNodes := map[string]*Node{}

	for name, edge := range edges {
		target := targets[name]
		edge.Dependencies = make([]*Node, 0, len(target.Dependencies))
		for _, v := range target.Dependencies {
			_, depName := splitManifestTarget(v)
			dep := edges[depName]
			edge.Dependencies = append(edge.Dependencies, dep)
			depNodes[dep.Name] = dep
		}

		edge.Configs = make([]*Node, 0, len(target.Configs))
		for _, v := range target.Configs {
			_, configName := splitManifestTarget(v)
			config := edges[configName]
			edge.Configs = append(edge.Configs, config)
		}
	}

	sourceNodes := []*Node{}

	for name, edge := range edges {
		if depNodes[name] != nil {
			// NOTE: This edge is not source.
			continue
		}
		sourceNodes = append(sourceNodes, edge)
	}

	graph := &Graph{
		edges:   edges,
		sources: sourceNodes,
	}
	return graph, nil
}
