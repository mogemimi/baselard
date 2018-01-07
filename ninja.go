package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type NinjaRule struct {
	Name        string
	Command     string
	Description string
	Deps        string
	DepFile     string
}

type NinjaBuild struct {
	Rule         string
	Outputs      []string
	Inputs       []string
	ImplicitOuts []string
	ImplicitDeps []string
	Variables    map[string]string
	Pool         string
}

type NinjaGenerator struct {
	Variables []string
	Rules     []*NinjaRule
	Edges     []*NinjaBuild
}

func (gen *NinjaGenerator) AddRule(rule *NinjaRule) {
	gen.Rules = append(gen.Rules, rule)
}

func (gen *NinjaGenerator) AddEdge(edge *NinjaBuild) {
	gen.Edges = append(gen.Edges, edge)
}

func (gen *NinjaGenerator) AddVariable(key, value string) {
	gen.Variables = append(gen.Variables, key+" = "+value)
}

func (r *NinjaRule) ToString() (str string) {
	str += fmt.Sprintln("rule", r.Name)
	if len(r.Description) > 0 {
		str += fmt.Sprintln("  description =", r.DepFile)
	}
	str += fmt.Sprintln("  command =", r.Command)
	if len(r.Deps) > 0 {
		str += fmt.Sprintln("  deps =", r.Deps)
	}
	if len(r.DepFile) > 0 {
		str += fmt.Sprintln("  depfile =", r.DepFile)
	}
	return str
}

func (e *NinjaBuild) ToString() (str string) {
	useMultiLine := (len(e.Outputs)+len(e.ImplicitOuts) > 1) || (len(e.Inputs)+len(e.ImplicitDeps) > 1)

	str += "build"
	if len(e.Outputs) > 0 {
		if useMultiLine {
			str += " $\n  "
		} else {
			str += " "
		}
	}
	for i, f := range e.Outputs {
		if i > 0 {
			str += " $\n  "
		}
		str += f
	}
	if len(e.ImplicitOuts) > 0 {
		str += " | "
		if useMultiLine {
			str += "$\n  "
		}
	}
	for i, f := range e.ImplicitOuts {
		if i > 0 {
			str += " $\n  "
		}
		str += f
	}
	str += ": "
	str += e.Rule
	if len(e.Inputs) > 0 {
		if useMultiLine {
			str += " $\n  "
		} else {
			str += " "
		}
	}
	for i, f := range e.Inputs {
		if i > 0 {
			str += " $\n  "
		}
		str += f
	}
	if len(e.ImplicitDeps) > 0 {
		str += " | "
		if useMultiLine {
			str += "$\n  "
		}
	}
	for i, f := range e.ImplicitDeps {
		if i > 0 {
			str += " $\n  "
		}
		str += f
	}
	str += "\n"
	if len(e.Pool) > 0 {
		str += fmt.Sprintf("  pool = %s\n", e.Pool)
	}
	var variables []string
	for k, v := range e.Variables {
		variables = append(variables, fmt.Sprintf("  %s = %s\n", k, v))
	}
	sort.Strings(variables)
	for _, v := range variables {
		str += v
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

		generator.AddEdge(&NinjaBuild{
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
		Deps:    "gcc",
		DepFile: "$out.d",
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
		generator.AddEdge(&NinjaBuild{
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
		generator.AddEdge(&NinjaBuild{
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
