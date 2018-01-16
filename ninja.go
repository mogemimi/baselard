package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

// NinjaGenerator generates a ninja file.
type NinjaGenerator struct {
	Variables []string
	Rules     []*NinjaRule
	Nodes     []*NinjaBuild
}

// AddRule adds the new rule to the ninja definition.
func (gen *NinjaGenerator) AddRule(rule *NinjaRule) {
	gen.Rules = append(gen.Rules, rule)
}

// AddNode adds the build node to the ninja graph.
func (gen *NinjaGenerator) AddNode(edge *NinjaBuild) {
	gen.Nodes = append(gen.Nodes, edge)
}

// AddVariable adds the new variable to the ninja definition.
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

	cflags := edge.GetCompilerFlags(env)
	cflagsC := edge.GetCompilerFlagsC(env)
	cflagsCC := edge.GetCompilerFlagsCC(env)

	for _, source := range sources {
		sourceFileType := getSourceFileType(source)

		obj := filepath.Clean(filepath.Join(env.OutDir, "obj", source+".o"))
		objFiles = append(objFiles, obj)

		variables := map[string]string{}
		if len(includeDirs) > 0 {
			variables["include_dirs"] = joinNinjaOptions("-I", includeDirs)
		}
		if len(defines) > 0 {
			variables["defines"] = joinNinjaOptions("-D", defines)
		}

		variables["cflags"] = strings.Join(cflags, " ")

		var compileRule string
		switch sourceFileType {
		case SourceFileTypeCSource:
			compileRule = "compile_c"
			variables["cflags_c"] = strings.Join(cflagsC, " ")
		case SourceFileTypeCppSource:
			compileRule = "compile"
			variables["cflags_cc"] = strings.Join(cflagsCC, " ")
		default:
			continue
		}

		generator.AddNode(&NinjaBuild{
			Rule:      compileRule,
			Inputs:    []string{source},
			Outputs:   []string{obj},
			Variables: variables,
		})
	}
	return objFiles
}

// Generate generates the ninja definitions from　graph contains the intermediate nodes.
func (gen *NinjaGenerator) Generate(env *Environment, edges map[string]*Node) {
	// $cxx -MMD -MF $out.d $defines $includes $cflags $cflags_cc
	gen.AddRule(&NinjaRule{
		Name:    "compile_c",
		Command: "clang -MMD -MF $out.d $defines $include_dirs $cflags $cflags_c -c $in -o $out",
		Deps:    "gcc",
		DepFile: "$out.d",
	})
	gen.AddRule(&NinjaRule{
		Name:    "compile",
		Command: "clang++ -MMD -MF $out.d $defines $include_dirs $cflags $cflags_cc -c $in -o $out",
		Deps:    "gcc",
		DepFile: "$out.d",
	})
	gen.AddRule(&NinjaRule{
		Name:    "link",
		Command: "ld $in $ldflags -o $out",
	})
	gen.AddRule(&NinjaRule{
		Name:    "archive",
		Command: "ar -rc $out $in",
	})
	gen.AddRule(&NinjaRule{
		Name:    "archive-static-libs",
		Command: "libtool -static -o $out $in",
	})

	for _, edge := range edges {
		switch edge.Type {
		case OutputTypeExecutable:
			objFiles := compileSources(env, edge, gen)
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
			gen.AddNode(&NinjaBuild{
				Rule:         "link",
				Inputs:       objFiles,
				ImplicitDeps: libraryFiles,
				Outputs:      []string{executableFile},
				Variables: map[string]string{
					"ldflags": strings.Join(ldflags, " "),
				},
			})
		case OutputTypeStaticLibrary:
			objFiles := compileSources(env, edge, gen)
			libraryFiles := []string{}
			for _, dep := range edge.Dependencies {
				switch dep.Type {
				case OutputTypeStaticLibrary:
					lib := filepath.Join(env.OutDir, "bin", "lib"+dep.Name+".a")
					libraryFiles = append(libraryFiles, lib)
				}
			}
			libFile := filepath.Join(env.OutDir, "bin", "lib"+edge.Name+".a")
			gen.AddNode(&NinjaBuild{
				Rule:    "archive-static-libs",
				Inputs:  append(objFiles, libraryFiles...),
				Outputs: []string{libFile},
			})
		}
	}
}

// WriteFile writes the ninja defintions to the specified file.
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
