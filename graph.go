package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Graph struct {
	edges   map[string]*Edge
	sources []*Edge
}

func parseGraph(manifestFile string) (*Graph, error) {
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
		manifest.Imports = normalizePathList(baseDir, manifest.Imports)

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
				Headers:       normalizePathList(baseDir, target.Headers),
				Sources:       normalizePathList(baseDir, target.Sources),
				IncludeDirs:   normalizePathList(baseDir, target.IncludeDirs),
				LibDirs:       normalizePathList(baseDir, target.LibDirs),
				Defines:       target.Defines,
				CompilerFlags: target.CompilerFlags,
				LinkerFlags:   target.LinkerFlags,
			}

			edge.Tagged = map[string]*Edge{}
			for tag, tagged := range target.Tagged {
				edge.Tagged[tag] = &Edge{
					Headers:       normalizePathList(baseDir, tagged.Headers),
					Sources:       normalizePathList(baseDir, tagged.Sources),
					IncludeDirs:   normalizePathList(baseDir, tagged.IncludeDirs),
					LibDirs:       normalizePathList(baseDir, tagged.LibDirs),
					Defines:       tagged.Defines,
					CompilerFlags: tagged.CompilerFlags,
					LinkerFlags:   tagged.LinkerFlags,
				}
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

	graph := &Graph{
		edges:   edges,
		sources: sourceEdges,
	}
	return graph, nil
}
