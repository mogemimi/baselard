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

func (edge *Edge) GetHeaders() (result []string) {
	result = append(result, edge.Headers...)
	for _, c := range edge.Configs {
		result = append(result, c.Headers...)
	}
	return result
}

func (edge *Edge) GetSources() (result []string) {
	result = append(result, edge.Sources...)
	for _, c := range edge.Configs {
		result = append(result, c.Sources...)
	}
	return result
}

func (edge *Edge) GetIncludeDirs() (result []string) {
	result = append(result, edge.IncludeDirs...)
	for _, c := range edge.Configs {
		result = append(result, c.IncludeDirs...)
	}
	return result
}

func (edge *Edge) GetLibDirs() (result []string) {
	result = append(result, edge.LibDirs...)
	for _, c := range edge.Configs {
		result = append(result, c.LibDirs...)
	}
	return result
}

func (edge *Edge) GetDefines() (result []string) {
	result = append(result, edge.Defines...)
	for _, c := range edge.Configs {
		result = append(result, c.Defines...)
	}
	return result
}

func (edge *Edge) GetCompilerFlags() (result []string) {
	result = append(result, edge.CompilerFlags...)
	for _, c := range edge.Configs {
		result = append(result, c.CompilerFlags...)
	}
	return result
}

func (edge *Edge) GetLinkerFlags() (result []string) {
	result = append(result, edge.LinkerFlags...)
	for _, c := range edge.Configs {
		result = append(result, c.LinkerFlags...)
	}
	return result
}
