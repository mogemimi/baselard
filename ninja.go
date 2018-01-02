package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type NinjaDepOption struct {
	Deps    string
	DepFile string
}

type NinjaRule struct {
	Name        string
	Command     string
	Description string // TODO
	DepOption   *NinjaDepOption
}

type BuildEdge struct {
	Rule      string
	Inputs    []string
	Outputs   []string
	Variables map[string]string
}

type NinjaGenerator struct {
	Variables []string
	Rules     []*NinjaRule
	Edges     []*BuildEdge
}

func (gen *NinjaGenerator) AddRule(rule *NinjaRule) {
	gen.Rules = append(gen.Rules, rule)
}

func (gen *NinjaGenerator) AddEdge(edge *BuildEdge) {
	gen.Edges = append(gen.Edges, edge)
}

func (gen *NinjaGenerator) AddVariable(key, value string) {
	gen.Variables = append(gen.Variables, key+" = "+value)
}

func (r *NinjaRule) ToString() (str string) {
	str += fmt.Sprintln("rule", r.Name)
	str += fmt.Sprintln("  command =", r.Command)
	if r.DepOption != nil {
		str += fmt.Sprintln("  deps =", r.DepOption.Deps)
		str += fmt.Sprintln("  depfile =", r.DepOption.DepFile)
	}
	return str
}

func (e *BuildEdge) ToString() (str string) {
	str += "build "
	for i, f := range e.Outputs {
		if i > 0 {
			str += " $\n  "
		}
		str += f
	}
	str += ": "
	str += e.Rule
	if len(e.Inputs) > 1 {
		str += " $\n  "
	} else {
		str += " "
	}
	for i, f := range e.Inputs {
		if i > 0 {
			str += " $\n  "
		}
		str += f
	}
	str += "\n"
	for k, v := range e.Variables {
		str += fmt.Sprintf("  %s = %s\n", k, v)
	}
	return str
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

func compileSources(env *Environment, executable *Executable, generator *NinjaGenerator) (objFiles []string) {
	for _, source := range executable.Sources {
		obj := filepath.Clean(filepath.Join(env.OutDir, "obj", source+".o"))
		objFiles = append(objFiles, obj)

		variables := map[string]string{}
		if len(executable.IncludeDirs) > 0 {
			variables["include_dirs"] = joinNinjaOptions("-I", executable.IncludeDirs)
		}
		if len(executable.Defines) > 0 {
			variables["defines"] = joinNinjaOptions("-D", executable.Defines)
		}

		cflags := []string{
			"-Wall",
			"-std=c++14",
			"-target x86_64-apple-macosx10.11",
		}
		variables["cflags"] = strings.Join(cflags, " ")

		generator.AddEdge(&BuildEdge{
			Rule:      "compile",
			Inputs:    []string{source},
			Outputs:   []string{obj},
			Variables: variables,
		})
	}
	return objFiles
}

func (generator *NinjaGenerator) Generate(env *Environment, conf *Config) {
	// $cxx -MMD -MF $out.d $defines $includes $cflags $cflags_cc
	generator.AddRule(&NinjaRule{
		Name:    "compile",
		Command: "clang -MMD -MF $out.d $defines $include_dirs $cflags -c $in -o $out",
		DepOption: &NinjaDepOption{
			Deps:    "gcc",
			DepFile: "$out.d",
		},
	})
	generator.AddRule(&NinjaRule{
		Name:    "link",
		Command: "ld $sources $ldflags -o $out",
	})
	generator.AddRule(&NinjaRule{
		Name:    "archive",
		Command: "ar -rc $out $in",
	})

	for _, executable := range conf.Executables {
		objFiles := compileSources(env, &executable, generator)
		libraryFiles := []string{}
		ldflags := []string{
			"-lSystem",
			"-lc++",
			"-macosx_version_min 10.11",
			"-L" + filepath.Join(env.OutDir, "bin"),
		}
		for _, dep := range executable.Dependencies {
			lib := filepath.Join(env.OutDir, "bin", "lib"+dep+".a")
			libraryFiles = append(libraryFiles, lib)
			ldflags = append(ldflags, "-l"+dep)
		}
		executableFile := filepath.Join(env.OutDir, "bin", executable.Name)
		generator.AddEdge(&BuildEdge{
			Rule:    "link",
			Inputs:  append(objFiles, libraryFiles...),
			Outputs: []string{executableFile},
			Variables: map[string]string{
				"sources": strings.Join(objFiles, " "),
				"ldflags": strings.Join(ldflags, " "),
			},
		})
	}
	for _, executable := range conf.StaticLibraries {
		objFiles := compileSources(env, &executable, generator)
		libFile := filepath.Join(env.OutDir, "bin", "lib"+executable.Name+".a")
		generator.AddEdge(&BuildEdge{
			Rule:    "archive",
			Inputs:  objFiles,
			Outputs: []string{libFile},
		})
	}
}

func (gen *NinjaGenerator) WriteFile(ninjaFile string) error {
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

	for _, edge := range gen.Edges {
		if _, err := writer.WriteString(edge.ToString()); err != nil {
			return err
		}
	}

	writer.Flush()

	return nil
}
