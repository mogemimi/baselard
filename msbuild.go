package main

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

type MSBuildGenerator struct {
	projects []MSBuildProjectSource
}

type MSBuildProjectSource struct {
	Name           string
	Project        MSBuildXMLProject
	ProjectFilters MSBuildXMLProject
}

type MSBuildXMLProject struct {
	XMLName        xml.Name `xml:"Project"`
	DefaultTargets string   `xml:"DefaultTargets,attr"`
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
	ProjectConfiguration []MSBuildXMLProjectConfiguration `xml:"ProjectConfiguration,omitempty"`
	Text                 []MSBuildXMLItem                 `xml:"Text"`
	ClInclude            []MSBuildXMLItem                 `xml:"ClInclude"`
	ClCompile            []MSBuildXMLItem                 `xml:"ClCompile"`
}

type MSBuildXMLItem struct {
	Include string `xml:"Include,attr"`
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

func (generator *MSBuildGenerator) Generate(env *Environment, graph *Graph, generatorSettings *GeneratorSettings) {
	for _, edge := range graph.edges {
		if edge.Type == OutputTypeUnknown {
			continue
		}

		projectSource := MSBuildProjectSource{}
		projectSource.Name = edge.Name

		project := edge.GetMSBuildProject(env)

		projectSource.Project = MSBuildXMLProject{
			DefaultTargets: "Build",
			ToolsVersion:   "14.0",
			Xmlns:          "http://schemas.microsoft.com/developer/msbuild/2003",
		}

		vcxproj := &projectSource.Project

		vcxproj.ItemGroup = append(vcxproj.ItemGroup, MSBuildXMLItemGroup{
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

		// TODO: Not implemented
		vcxproj.PropertyGroup = append(vcxproj.PropertyGroup, MSBuildXMLPropertyGroup{
			Name: "PropertyGroup",
			Attrs: map[string]string{
				"Label": "Globals",
			},
			Elements: map[string]string{
				"ProjectGuid":                  "B224C8A5-3C1A-4611-8372-6B52775D5B09",
				"Keyword":                      "Win32Proj",
				"RootNamespace":                "MyConsoleApplication1",
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

		for _, config := range project.Configurations {
			projectEnv := &Environment{}
			projectEnv.OutDir = env.OutDir
			projectEnv.Tags = env.Tags
			projectEnv.Tags = append(projectEnv.Tags, config.Tags...)

			msbuild := edge.GetMSBuildSettings(projectEnv)

			propertyGroupsConfiguration = append(propertyGroupsConfiguration, MSBuildXMLPropertyGroup{
				Name: "PropertyGroup",
				Attrs: map[string]string{
					"Condition": fmt.Sprintf("'$(Configuration)|$(Platform)'=='%s|%s'", config.Configuration, config.Platform),
					"Label":     "Configuration",
				},
				Elements: copyStringMap(msbuild.Configuration),
			})

			propertyGroupsGeneral = append(propertyGroupsGeneral, MSBuildXMLPropertyGroup{
				Name: "PropertyGroup",
				Attrs: map[string]string{
					"Condition": fmt.Sprintf("'$(Configuration)|$(Platform)'=='%s|%s'", config.Configuration, config.Platform),
				},
				Elements: copyStringMap(msbuild.General),
			})

			vcxproj.ItemDefinitionGroup = append(vcxproj.ItemDefinitionGroup, MSBuildXMLItemDefinitionGroup{
				Condition: fmt.Sprintf("'$(Configuration)|$(Platform)'=='%s|%s'", config.Configuration, config.Platform),
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

		vcxproj.ItemGroup = append(vcxproj.ItemGroup, MSBuildXMLItemGroup{
			ClInclude: func() (result []MSBuildXMLItem) {
				for _, s := range edge.GetHeaders(env) {
					result = append(result, MSBuildXMLItem{Include: s})
				}
				return result
			}(),
		})
		vcxproj.ItemGroup = append(vcxproj.ItemGroup, MSBuildXMLItemGroup{
			ClCompile: func() (result []MSBuildXMLItem) {
				for _, s := range edge.GetSources(env) {
					result = append(result, MSBuildXMLItem{Include: s})
				}
				return result
			}(),
		})

		generator.projects = append(generator.projects, projectSource)
	}
}

func (gen *MSBuildGenerator) WriteFile(outputDir string) error {
	for _, projectSource := range gen.projects {
		outputPath := filepath.Join(outputDir, projectSource.Name+".vcxproj")

		dir := filepath.Dir(outputPath)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err := os.MkdirAll(dir, os.ModePerm); err != nil {
				return errors.Wrapf(err, "Failed to create output directory \"%s\"", dir)
			}
		}

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

	return nil
}
