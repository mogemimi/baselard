package main

type OutputType int

const (
	OutputTypeUnknown OutputType = iota
	OutputTypeExecutable
	OutputTypeStaticLibrary
)

type Edge struct {
	Name          string
	Type          OutputType
	Headers       []string
	Sources       []string
	IncludeDirs   []string
	LibDirs       []string
	Defines       []string
	CompilerFlags []string
	LinkerFlags   []string
	Dependencies  []*Edge
	Configs       []*Edge
	Tagged        map[string]*Edge
}

func (edge *Edge) GetHeaders(env *Environment) (result []string) {
	result = append(result, edge.Headers...)
	for _, tag := range env.Tags {
		if tagged := edge.Tagged[tag]; tagged != nil {
			result = append(result, tagged.Headers...)
		}
	}
	for _, c := range edge.Configs {
		result = append(result, c.GetHeaders(env)...)
	}
	return result
}

func (edge *Edge) GetSources(env *Environment) (result []string) {
	result = append(result, edge.Sources...)
	for _, tag := range env.Tags {
		if tagged := edge.Tagged[tag]; tagged != nil {
			result = append(result, tagged.Sources...)
		}
	}
	for _, c := range edge.Configs {
		result = append(result, c.GetSources(env)...)
	}
	return result
}

func (edge *Edge) GetIncludeDirs(env *Environment) (result []string) {
	result = append(result, edge.IncludeDirs...)
	for _, tag := range env.Tags {
		if tagged := edge.Tagged[tag]; tagged != nil {
			result = append(result, tagged.IncludeDirs...)
		}
	}
	for _, c := range edge.Configs {
		result = append(result, c.GetIncludeDirs(env)...)
	}
	return result
}

func (edge *Edge) GetLibDirs(env *Environment) (result []string) {
	result = append(result, edge.LibDirs...)
	for _, tag := range env.Tags {
		if tagged := edge.Tagged[tag]; tagged != nil {
			result = append(result, tagged.LibDirs...)
		}
	}
	for _, c := range edge.Configs {
		result = append(result, c.GetLibDirs(env)...)
	}
	return result
}

func (edge *Edge) GetDefines(env *Environment) (result []string) {
	result = append(result, edge.Defines...)
	for _, tag := range env.Tags {
		if tagged := edge.Tagged[tag]; tagged != nil {
			result = append(result, tagged.Defines...)
		}
	}
	for _, c := range edge.Configs {
		result = append(result, c.GetDefines(env)...)
	}
	return result
}

func (edge *Edge) GetCompilerFlags(env *Environment) (result []string) {
	result = append(result, edge.CompilerFlags...)
	for _, tag := range env.Tags {
		if tagged := edge.Tagged[tag]; tagged != nil {
			result = append(result, tagged.CompilerFlags...)
		}
	}
	for _, c := range edge.Configs {
		result = append(result, c.GetCompilerFlags(env)...)
	}
	return result
}

func (edge *Edge) GetLinkerFlags(env *Environment) (result []string) {
	result = append(result, edge.LinkerFlags...)
	for _, tag := range env.Tags {
		if tagged := edge.Tagged[tag]; tagged != nil {
			result = append(result, tagged.LinkerFlags...)
		}
	}
	for _, c := range edge.Configs {
		result = append(result, c.GetLinkerFlags(env)...)
	}
	return result
}
