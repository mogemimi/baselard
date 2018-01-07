package main

import (
	"fmt"
	"sort"
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
		if len(e.Outputs)+len(e.ImplicitOuts) > 1 {
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
