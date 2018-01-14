package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

type NinjaGenerator struct {
	Variables []string
	Rules     []*NinjaRule
	Nodes     []*NinjaBuild
}

func (gen *NinjaGenerator) AddRule(rule *NinjaRule) {
	gen.Rules = append(gen.Rules, rule)
}

func (gen *NinjaGenerator) AddNode(edge *NinjaBuild) {
	gen.Nodes = append(gen.Nodes, edge)
}

func (gen *NinjaGenerator) AddVariable(key, value string) {
	gen.Variables = append(gen.Variables, key+" = "+value)
}

func joinNinjaOptions(prefix string, options []string) string {
	str := ""
	for i, d := range options {
		if i > 0 {
			str += " "
		}
		str += prefix
		str += d
	}
	return str
}

func compileSources(env *Environment, edge *Node, generator *NinjaGenerator) (objFiles []string) {
	sources := edge.GetSources(env)
	includeDirs := edge.GetIncludeDirs(env)
	defines := edge.GetDefines(env)

	for _, source := range sources {
		obj := filepath.Clean(filepath.Join(env.OutDir, "obj", source+".o"))
		objFiles = append(objFiles, obj)

		variables := map[string]string{}
		if len(includeDirs) > 0 {
			variables["include_dirs"] = joinNinjaOptions("-I", includeDirs)
		}
		if len(defines) > 0 {
			variables["defines"] = joinNinjaOptions("-D", defines)
		}

		cflags := []string{}
		cflags = append(cflags, edge.GetCompilerFlags(env)...)
		variables["cflags"] = strings.Join(cflags, " ")

		generator.AddNode(&NinjaBuild{
			Rule:      "compile",
			Inputs:    []string{source},
			Outputs:   []string{obj},
			Variables: variables,
		})
	}
	return objFiles
}

func (generator *NinjaGenerator) Generate(env *Environment, edges map[string]*Node) {
	// $cxx -MMD -MF $out.d $defines $includes $cflags $cflags_cc
	generator.AddRule(&NinjaRule{
		Name:    "compile",
		Command: "clang -MMD -MF $out.d $defines $include_dirs $cflags -c $in -o $out",
		Deps:    "gcc",
		DepFile: "$out.d",
	})
	generator.AddRule(&NinjaRule{
		Name:    "link",
		Command: "ld $in $ldflags -o $out",
	})
	generator.AddRule(&NinjaRule{
		Name:    "archive",
		Command: "ar -rc $out $in",
	})
	generator.AddRule(&NinjaRule{
		Name:    "archive-static-libs",
		Command: "libtool -static -o $out $in",
	})

	for _, edge := range edges {
		switch edge.Type {
		case OutputTypeExecutable:
			objFiles := compileSources(env, edge, generator)
			libraryFiles := []string{}
			ldflags := []string{
				"-L" + filepath.Join(env.OutDir, "bin"),
			}
			for _, f := range edge.GetLinkerFlags(env) {
				ldflags = append(ldflags, f)
			}
			for _, dir := range edge.GetLibDirs(env) {
				ldflags = append(ldflags, "-L"+dir)
			}
			for _, dep := range edge.Dependencies {
				switch dep.Type {
				case OutputTypeStaticLibrary:
					lib := filepath.Join(env.OutDir, "bin", "lib"+dep.Name+".a")
					libraryFiles = append(libraryFiles, lib)
					ldflags = append(ldflags, "-l"+dep.Name)
				}
			}
			executableFile := filepath.Join(env.OutDir, "bin", edge.Name)
			generator.AddNode(&NinjaBuild{
				Rule:         "link",
				Inputs:       objFiles,
				ImplicitDeps: libraryFiles,
				Outputs:      []string{executableFile},
				Variables: map[string]string{
					"ldflags": strings.Join(ldflags, " "),
				},
			})
		case OutputTypeStaticLibrary:
			objFiles := compileSources(env, edge, generator)
			libraryFiles := []string{}
			for _, dep := range edge.Dependencies {
				switch dep.Type {
				case OutputTypeStaticLibrary:
					lib := filepath.Join(env.OutDir, "bin", "lib"+dep.Name+".a")
					libraryFiles = append(libraryFiles, lib)
				}
			}
			libFile := filepath.Join(env.OutDir, "bin", "lib"+edge.Name+".a")
			generator.AddNode(&NinjaBuild{
				Rule:    "archive-static-libs",
				Inputs:  append(objFiles, libraryFiles...),
				Outputs: []string{libFile},
			})
		}
	}
}

func (gen *NinjaGenerator) WriteFile(ninjaFile string) error {
	dir := filepath.Dir(ninjaFile)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return errors.Wrapf(err, "Failed to create output directory \"%s\"", dir)
		}
	}

	file, err := os.Create(ninjaFile)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	for _, v := range gen.Variables {
		if _, err := writer.WriteString(v + "\n"); err != nil {
			return err
		}
	}
	if len(gen.Variables) > 0 {
		if _, err := writer.WriteString("\n"); err != nil {
			return err
		}
	}

	for i, rule := range gen.Rules {
		if i > 0 {
			if _, err := writer.WriteString("\n"); err != nil {
				return err
			}
		}
		if _, err := writer.WriteString(rule.ToString()); err != nil {
			return err
		}
	}

	if len(gen.Rules) > 0 {
		if _, err := writer.WriteString("\n"); err != nil {
			return err
		}
	}

	for _, edge := range gen.Nodes {
		if _, err := writer.WriteString(edge.ToString()); err != nil {
			return err
		}
	}

	writer.Flush()

	return nil
}
