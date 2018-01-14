package main

import (
	"fmt"
)

type OutputType int

const (
	OutputTypeUnknown OutputType = iota
	OutputTypeExecutable
	OutputTypeStaticLibrary
	OutputTypeDynamicLibrary
)

type Node struct {
	Name            string
	Type            OutputType
	Headers         []string
	Sources         []string
	IncludeDirs     []string
	LibDirs         []string
	Defines         []string
	CompilerFlags   []string
	LinkerFlags     []string
	MSBuildSettings MSBuildSettings
	MSBuildProject  MSBuildProject
	Dependencies    []*Node
	Configs         []*Node
	Tagged          map[string]*Node
}

func (edge *Node) GetHeaders(env *Environment) (result []string) {
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

func (edge *Node) GetSources(env *Environment) (result []string) {
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

func (edge *Node) GetIncludeDirs(env *Environment) (result []string) {
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

func (edge *Node) GetLibDirs(env *Environment) (result []string) {
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

func (edge *Node) GetDefines(env *Environment) (result []string) {
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

func (edge *Node) GetCompilerFlags(env *Environment) (result []string) {
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

func (edge *Node) GetLinkerFlags(env *Environment) (result []string) {
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

func copyMSBuildProjectConfiguration(dst, src *MSBuildProjectConfiguration) {
	dst.Configuration = src.Configuration
	dst.Platform = src.Platform
	dst.ExecutableExtension = src.ExecutableExtension
	dst.StaticLibraryExtension = src.StaticLibraryExtension
	dst.DynamicLibraryExtension = src.DynamicLibraryExtension
	dst.Tags = make([]string, len(src.Tags))
	copy(dst.Tags, src.Tags)
}

func (edge *Node) GetMSBuildProject(env *Environment) MSBuildProject {
	result := MSBuildProject{}
	for _, v := range edge.MSBuildProject.Configurations {
		c := MSBuildProjectConfiguration{}
		copyMSBuildProjectConfiguration(&c, &v)
		result.Configurations = append(result.Configurations, c)
	}
	result.ExtensionSettings = append(result.ExtensionSettings, edge.MSBuildProject.ExtensionSettings...)
	result.ExtensionTargets = append(result.ExtensionTargets, edge.MSBuildProject.ExtensionTargets...)

	configs := map[string]*MSBuildProjectConfiguration{}
	for _, v := range result.Configurations {
		key := fmt.Sprintln(v.Configuration, "|", v.Platform)
		configs[key] = &v
	}

	for _, c := range edge.Configs {
		other := c.GetMSBuildProject(env)
		for _, v := range other.Configurations {
			key := fmt.Sprintln(v.Configuration, "|", v.Platform)
			if conf, ok := configs[key]; ok {
				conf.Tags = append(conf.Tags, v.Tags...)
			} else {
				c := MSBuildProjectConfiguration{}
				copyMSBuildProjectConfiguration(&c, &v)
				result.Configurations = append(result.Configurations, c)
			}
		}
		result.ExtensionSettings = append(result.ExtensionSettings, other.ExtensionSettings...)
		result.ExtensionTargets = append(result.ExtensionTargets, other.ExtensionTargets...)
	}
	return result
}

func mergeMSBuildSettingsMap(a, b *map[string]string) {
	if (*b) == nil {
		return
	}
	for k, v := range *b {
		if _, ok := (*a)[k]; !ok {
			(*a)[k] = v
		}
	}
}

func mergeMSBuildSettings(a, b *MSBuildSettings) {
	mergeMSBuildSettingsMap(&a.ClCompile, &b.ClCompile)
	mergeMSBuildSettingsMap(&a.Link, &b.Link)
	mergeMSBuildSettingsMap(&a.Lib, &b.Lib)
	mergeMSBuildSettingsMap(&a.Globals, &b.Globals)
	mergeMSBuildSettingsMap(&a.Configuration, &b.Configuration)
	mergeMSBuildSettingsMap(&a.User, &b.User)
	mergeMSBuildSettingsMap(&a.General, &b.General)
}

func copyStringMap(s map[string]string) map[string]string {
	result := make(map[string]string)
	for k, v := range s {
		result[k] = v
	}
	return result
}

func copyMSBuildSettings(dst, src *MSBuildSettings) {
	dst.ClCompile = copyStringMap(src.ClCompile)
	dst.Link = copyStringMap(src.Link)
	dst.Lib = copyStringMap(src.Lib)
	dst.Globals = copyStringMap(src.Globals)
	dst.User = copyStringMap(src.User)
	dst.General = copyStringMap(src.General)
	dst.Configuration = copyStringMap(src.Configuration)
}

func (edge *Node) GetMSBuildSettings(env *Environment) MSBuildSettings {
	result := MSBuildSettings{}
	copyMSBuildSettings(&result, &edge.MSBuildSettings)

	for _, tag := range env.Tags {
		if tagged, ok := edge.Tagged[tag]; ok && tagged != nil {
			mergeMSBuildSettings(&result, &tagged.MSBuildSettings)
		}
	}

	for _, c := range edge.Configs {
		other := c.GetMSBuildSettings(env)
		mergeMSBuildSettings(&result, &other)
	}
	return result
}