package main

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
)

type MSBuildGenerator struct {
	projects []*MSBuildProjectSource
}

type MSBuildProjectSource struct {
	Name           string
	GUID           string
	FilePath       string
	Conditions     []string
	DependProjects []string
	Project        MSBuildXMLProject
	ProjectFilters MSBuildXMLProject
}

type MSBuildXMLProject struct {
	XMLName        xml.Name `xml:"Project"`
	DefaultTargets string   `xml:"DefaultTargets,attr,omitempty"`
	ToolsVersion   string   `xml:"ToolsVersion,attr"`
	Xmlns          string   `xml:"xmlns,attr"`

	ItemGroup           []MSBuildXMLItemGroup           `xml:"ItemGroup"`
	PropertyGroup       []MSBuildXMLPropertyGroup       `xml:"PropertyGroup"`
	ImportGroup         []MSBuildXMLImportGroup         `xml:"ImportGroup"`
	Import              []MSBuildXMLImport              `xml:"Import"`
	ItemDefinitionGroup []MSBuildXMLItemDefinitionGroup `xml:"ItemDefinitionGroup"`
}

type MSBuildXMLItemGroup struct {
	XMLName              xml.Name                         `xml:"ItemGroup"`
	Label                string                           `xml:"Label,attr,omitempty"`
	Filter               []MSBuildXMLFilter               `xml:"Filter,omitempty"`
	ProjectConfiguration []MSBuildXMLProjectConfiguration `xml:"ProjectConfiguration,omitempty"`
	Text                 []MSBuildXMLItem                 `xml:"Text"`
	ClInclude            []MSBuildXMLItem                 `xml:"ClInclude"`
	ClCompile            []MSBuildXMLItem                 `xml:"ClCompile"`
}

type MSBuildXMLFilter struct {
	Include          string `xml:"Include,attr,omitempty"`
	UniqueIdentifier string `xml:"UniqueIdentifier"`
	Extensions       string `xml:"Extensions"`
}

type MSBuildXMLExcludedFromBuild struct {
	Condition string `xml:"Condition,attr,omitempty"`
	Excluded  bool   `xml:",chardata"`
}

type MSBuildXMLItem struct {
	Include           string                        `xml:"Include,attr"`
	ExcludedFromBuild []MSBuildXMLExcludedFromBuild `xml:"ExcludedFromBuild,omitempty"`
	Filter            string                        `xml:"Filter,omitempty"`
}

type MSBuildXMLProjectConfiguration struct {
	Include       string `xml:"Include,attr"`
	Configuration string `xml:"Configuration,omitempty"`
	Platform      string `xml:"Platform,omitempty"`
}

type MSBuildXMLPropertyGroup struct {
	Name     string
	Attrs    map[string]string
	Elements map[string]string
}

func (u MSBuildXMLPropertyGroup) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = u.Name
	start.Attr = []xml.Attr{}
	for k, v := range u.Attrs {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: k}, Value: v})
	}
	e.EncodeToken(start)
	for k, v := range u.Elements {
		e.EncodeElement(v, xml.StartElement{Name: xml.Name{Local: k}})
	}
	e.EncodeToken(start.End())
	return nil
}

type MSBuildXMLItemDefinitionGroup struct {
	Condition       string                    `xml:"Condition,attr,omitempty"`
	ItemDefinitions []MSBuildXMLPropertyGroup `xml:",omitempty"`
}

type MSBuildXMLImport struct {
	Project   string `xml:"Project,attr"`
	Condition string `xml:"Condition,attr,omitempty"`
	Label     string `xml:"Label,attr,omitempty"`
}

type MSBuildXMLImportGroup struct {
	Condition string             `xml:"Condition,attr,omitempty"`
	Label     string             `xml:"Label,attr,omitempty"`
	Import    []MSBuildXMLImport `xml:"Import"`
}

func getClCompileSources(edge *Edge, project *MSBuildProject, env *Environment) (result []MSBuildXMLItem) {
	type SourceConditions struct {
		Conditions map[string]bool
	}
	sources := map[string]SourceConditions{}

	projectConditions := []string{}

	for _, config := range project.Configurations {
		projectEnv := &Environment{}
		projectEnv.OutDir = env.OutDir
		projectEnv.Tags = env.Tags
		projectEnv.Tags = append(projectEnv.Tags, config.Tags...)

		condition := fmt.Sprintf("'$(Configuration)|$(Platform)'=='%s|%s'", config.Configuration, config.Platform)

		projectConditions = append(projectConditions, condition)

		for _, src := range edge.GetSources(projectEnv) {
			if _, ok := sources[src]; !ok {
				sources[src] = SourceConditions{
					Conditions: map[string]bool{},
				}
			}
			sources[src].Conditions[condition] = true
		}
	}

	for src, conditions := range sources {
		src, _ = filepath.Rel(env.OutDir, src)
		item := MSBuildXMLItem{Include: src}

		if len(project.Configurations) > len(conditions.Conditions) {
			for cond := range conditions.Conditions {
				item.ExcludedFromBuild = append(item.ExcludedFromBuild, MSBuildXMLExcludedFromBuild{
					Condition: cond,
					Excluded:  false,
				})
			}
			for _, c := range projectConditions {
				if _, ok := conditions.Conditions[c]; !ok {
					item.ExcludedFromBuild = append(item.ExcludedFromBuild, MSBuildXMLExcludedFromBuild{
						Condition: c,
						Excluded:  true,
					})
				}
			}
		}
		result = append(result, item)
	}

	return result
}

func (generator *MSBuildGenerator) Generate(env *Environment, graph *Graph, generatorSettings *GeneratorSettings) {

	projectSourceMap := map[*Edge]*MSBuildProjectSource{}
	for _, edge := range graph.edges {
		if edge.Type == OutputTypeUnknown {
			continue
		}

		projectSource := &MSBuildProjectSource{
			Name:     edge.Name,
			GUID:     strings.ToUpper(uuid.NewV4().String()),
			FilePath: filepath.Join(env.ProjectFileDir, edge.Name+".vcxproj"),
		}

		projectSourceMap[edge] = projectSource
		generator.projects = append(generator.projects, projectSource)
	}

	for _, edge := range graph.edges {
		projectSource := projectSourceMap[edge]

		for _, dep := range edge.Dependencies {
			if depProject, ok := projectSourceMap[dep]; ok {
				projectSource.DependProjects = append(projectSource.DependProjects, depProject.GUID)
			}
		}
	}

	for _, edge := range graph.edges {
		if edge.Type == OutputTypeUnknown {
			continue
		}

		projectSource := projectSourceMap[edge]

		project := edge.GetMSBuildProject(env)

		projectSource.Project = MSBuildXMLProject{
			DefaultTargets: "Build",
			ToolsVersion:   "14.0",
			Xmlns:          "http://schemas.microsoft.com/developer/msbuild/2003",
		}

		vcxproj := &projectSource.Project

		vcxproj.ItemGroup = append(vcxproj.ItemGroup, MSBuildXMLItemGroup{
			Label: "ProjectConfigurations",
			ProjectConfiguration: func() (result []MSBuildXMLProjectConfiguration) {
				for _, v := range project.Configurations {
					result = append(result, MSBuildXMLProjectConfiguration{
						Include:       fmt.Sprintf("%s|%s", v.Configuration, v.Platform),
						Platform:      v.Platform,
						Configuration: v.Configuration,
					})
				}
				return result
			}(),
		})

		vcxproj.PropertyGroup = append(vcxproj.PropertyGroup, MSBuildXMLPropertyGroup{
			Name: "PropertyGroup",
			Attrs: map[string]string{
				"Label": "Globals",
			},
			Elements: map[string]string{
				"ProjectGuid":                  fmt.Sprintf("{%s}", projectSource.GUID),
				"Keyword":                      "Win32Proj",
				"RootNamespace":                projectSource.Name,
				"WindowsTargetPlatformVersion": "8.1",
			},
		})

		vcxproj.Import = append(vcxproj.Import, MSBuildXMLImport{
			Project: `$(VCTargetsPath)\Microsoft.Cpp.Default.props`,
		})
		vcxproj.Import = append(vcxproj.Import, MSBuildXMLImport{
			Project: `$(VCTargetsPath)\Microsoft.Cpp.props`,
		})
		vcxproj.Import = append(vcxproj.Import, MSBuildXMLImport{
			Project: `$(VCTargetsPath)\Microsoft.Cpp.targets`,
		})

		vcxproj.ImportGroup = append(vcxproj.ImportGroup, MSBuildXMLImportGroup{
			Label: "ExtensionSettings",
		})
		vcxproj.ImportGroup = append(vcxproj.ImportGroup, MSBuildXMLImportGroup{
			Label: "Shared",
		})

		for _, v := range project.Configurations {
			vcxproj.ImportGroup = append(vcxproj.ImportGroup, MSBuildXMLImportGroup{
				Label:     "PropertySheets",
				Condition: fmt.Sprintf("'$(Configuration)|$(Platform)'=='%s|%s'", v.Configuration, v.Platform),
				Import: []MSBuildXMLImport{
					{
						Project:   `$(UserRootDir)\Microsoft.Cpp.$(Platform).user.props`,
						Condition: `exists('$(UserRootDir)\Microsoft.Cpp.$(Platform).user.props')`,
						Label:     "LocalAppDataPlatform",
					},
				},
			})
		}

		vcxproj.PropertyGroup = append(vcxproj.PropertyGroup, MSBuildXMLPropertyGroup{
			Name: "PropertyGroup",
			Attrs: map[string]string{
				"Label": "UserMacros",
			},
		})

		var propertyGroupsConfiguration []MSBuildXMLPropertyGroup
		var propertyGroupsGeneral []MSBuildXMLPropertyGroup

		configurationType := func() string {
			switch edge.Type {
			case OutputTypeExecutable:
				return "Application"
			case OutputTypeStaticLibrary:
				return "StaticLibrary"
			case OutputTypeDynamicLibrary:
				return "DynamicLibrary"
			}
			return "Application"
		}()

		for _, config := range project.Configurations {
			projectEnv := &Environment{}
			projectEnv.OutDir = env.OutDir
			projectEnv.Tags = env.Tags
			projectEnv.Tags = append(projectEnv.Tags, config.Tags...)

			msbuild := edge.GetMSBuildSettings(projectEnv)

			msbuild.Configuration["ConfigurationType"] = configurationType

			msbuild.ClCompile["AdditionalIncludeDirectories"] = func() string {
				str := ""
				for _, dir := range edge.GetIncludeDirs(projectEnv) {
					dir, _ = filepath.Rel(env.OutDir, dir)
					str += dir
					str += ";"
				}
				str += "%(AdditionalIncludeDirectories)"
				return str
			}()

			msbuild.ClCompile["PreprocessorDefinitions"] = func() string {
				str := ""
				for _, def := range edge.GetDefines(projectEnv) {
					str += def
					str += ";"
				}
				str += "%(PreprocessorDefinitions)"
				return str
			}()

			msbuildLinker := func() map[string]string {
				switch edge.Type {
				case OutputTypeExecutable:
					return msbuild.Link
				case OutputTypeStaticLibrary:
					return msbuild.Lib
				}
				return msbuild.Link
			}()

			msbuildLinker["AdditionalLibraryDirectories"] = func() string {
				str := ""
				for _, dir := range edge.GetLibDirs(projectEnv) {
					dir, _ = filepath.Rel(env.OutDir, dir)
					str += dir
					str += ";"
				}
				str += "$(OutDir);"
				str += "%(AdditionalLibraryDirectories)"
				return str
			}()

			msbuildLinker["AdditionalDependencies"] = func() string {
				str := ""
				for _, dep := range edge.Dependencies {
					if dep.Type == OutputTypeStaticLibrary {
						str += (dep.Name + ".lib")
						str += ";"
					}
				}
				str += "%(AdditionalDependencies)"
				return str
			}()

			conditionStr := fmt.Sprintf("%s|%s", config.Configuration, config.Platform)
			projectSource.Conditions = append(projectSource.Conditions, conditionStr)

			propertyGroupsConfiguration = append(propertyGroupsConfiguration, MSBuildXMLPropertyGroup{
				Name: "PropertyGroup",
				Attrs: map[string]string{
					"Condition": fmt.Sprintf("'$(Configuration)|$(Platform)'=='%s'", conditionStr),
					"Label":     "Configuration",
				},
				Elements: copyStringMap(msbuild.Configuration),
			})

			propertyGroupsGeneral = append(propertyGroupsGeneral, MSBuildXMLPropertyGroup{
				Name: "PropertyGroup",
				Attrs: map[string]string{
					"Condition": fmt.Sprintf("'$(Configuration)|$(Platform)'=='%s'", conditionStr),
				},
				Elements: copyStringMap(msbuild.General),
			})

			vcxproj.ItemDefinitionGroup = append(vcxproj.ItemDefinitionGroup, MSBuildXMLItemDefinitionGroup{
				Condition: fmt.Sprintf("'$(Configuration)|$(Platform)'=='%s'", conditionStr),
				ItemDefinitions: func() (result []MSBuildXMLPropertyGroup) {
					if len(msbuild.ClCompile) > 0 {
						result = append(result, MSBuildXMLPropertyGroup{
							Name:     "ClCompile",
							Elements: copyStringMap(msbuild.ClCompile),
						})
					}
					if len(msbuild.Link) > 0 {
						result = append(result, MSBuildXMLPropertyGroup{
							Name:     "Link",
							Elements: copyStringMap(msbuild.Link),
						})
					}
					if len(msbuild.Lib) > 0 {
						result = append(result, MSBuildXMLPropertyGroup{
							Name:     "Lib",
							Elements: copyStringMap(msbuild.Lib),
						})
					}
					return result
				}(),
			})
		}

		vcxproj.PropertyGroup = append(vcxproj.PropertyGroup, propertyGroupsConfiguration...)
		vcxproj.PropertyGroup = append(vcxproj.PropertyGroup, propertyGroupsGeneral...)

		clIncludeSources := func() (result []MSBuildXMLItem) {
			for _, src := range edge.GetHeaders(env) {
				src, _ = filepath.Rel(env.OutDir, src)
				result = append(result, MSBuildXMLItem{Include: src})
			}
			return result
		}()
		vcxproj.ItemGroup = append(vcxproj.ItemGroup, MSBuildXMLItemGroup{
			ClInclude: clIncludeSources,
		})

		clCompileSources := getClCompileSources(edge, &project, env)
		vcxproj.ItemGroup = append(vcxproj.ItemGroup, MSBuildXMLItemGroup{
			ClCompile: clCompileSources,
		})

		projectSource.ProjectFilters = MSBuildXMLProject{
			ToolsVersion: "4.0",
			Xmlns:        "http://schemas.microsoft.com/developer/msbuild/2003",
		}

		filters := &projectSource.ProjectFilters
		filters.ItemGroup = append(filters.ItemGroup, MSBuildXMLItemGroup{
			Filter: []MSBuildXMLFilter{
				{
					Include:          "Source Files",
					UniqueIdentifier: "{4FC737F1-C7A5-4376-A066-2A32D752A2FF}",
					Extensions:       "cpp;c;cc;cxx;def;odl;idl;hpj;bat;asm;asmx",
				},
				{
					Include:          "Header Files",
					UniqueIdentifier: "{93995380-89BD-4b04-88EB-625FBE52EBFB}",
					Extensions:       "h;hh;hpp;hxx;hm;inl;inc;xsd",
				},
			},
		})

		for _, s := range clIncludeSources {
			filters.ItemGroup = append(filters.ItemGroup, MSBuildXMLItemGroup{
				ClInclude: []MSBuildXMLItem{{Include: s.Include, Filter: "Graphics"}},
			})
		}

		for _, s := range clCompileSources {
			filters.ItemGroup = append(filters.ItemGroup, MSBuildXMLItemGroup{
				ClCompile: []MSBuildXMLItem{{Include: s.Include, Filter: "Graphics"}},
			})
		}
	}
}

func (gen *MSBuildGenerator) WriteFile(env *Environment) error {
	solution := &MSBuildSolution{
		Name: "out",
	}

	for _, projectSource := range gen.projects {
		outputPath := projectSource.FilePath

		dir := filepath.Dir(outputPath)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err := os.MkdirAll(dir, os.ModePerm); err != nil {
				return errors.Wrapf(err, "Failed to create output directory \"%s\"", dir)
			}
		}

		{
			file, err := os.Create(outputPath)
			if err != nil {
				return err
			}
			defer file.Close()

			xmlString, err := xml.MarshalIndent(projectSource.Project, "", "  ")
			if err != nil {
				return err
			}

			// TODO: The following solution is too bad.
			replacedXML := strings.Replace(string(xmlString), "&#39;", "'", -1)

			writer := bufio.NewWriter(file)
			writer.WriteString(xml.Header)
			writer.WriteString(replacedXML)

			writer.Flush()
		}
		{
			outputPath += ".filters"

			file, err := os.Create(outputPath)
			if err != nil {
				return err
			}
			defer file.Close()

			xmlString, err := xml.MarshalIndent(projectSource.ProjectFilters, "", "  ")
			if err != nil {
				return err
			}

			// TODO: The following solution is too bad.
			replacedXML := strings.Replace(string(xmlString), "&#39;", "'", -1)

			writer := bufio.NewWriter(file)
			writer.WriteString(xml.Header)
			writer.WriteString(replacedXML)

			writer.Flush()
		}

		solution.Projects = append(solution.Projects, projectSource)
	}

	generateMSBuildSolutionFile(env, solution)

	return nil
}
