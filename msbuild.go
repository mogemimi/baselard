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
	Project        MSBuildXMLElement
	ProjectFilters MSBuildXMLElement
}

type MSBuildXMLExcludedFromBuild struct {
	Condition string
	Excluded  bool
}

type MSBuildXMLItem struct {
	Include           string
	ExcludedFromBuild []MSBuildXMLExcludedFromBuild
	Filter            string
}

type MSBuildXMLElement struct {
	Name       string
	Text       string
	Attributes []MSBuildXMLAttribute
	Elements   []*MSBuildXMLElement
}

type MSBuildXMLAttribute struct {
	Key   string
	Value string
}

func xmlAttr(key, value string) MSBuildXMLAttribute {
	return MSBuildXMLAttribute{Key: key, Value: value}
}

func (proj *MSBuildXMLElement) SubElement(name string, attributes ...MSBuildXMLAttribute) *MSBuildXMLElement {
	elem := &MSBuildXMLElement{
		Name:       name,
		Attributes: attributes,
	}
	proj.Elements = append(proj.Elements, elem)
	return elem
}

func (proj *MSBuildXMLElement) SetText(text string) *MSBuildXMLElement {
	proj.Text = text
	return proj
}

func (u MSBuildXMLElement) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = u.Name
	start.Attr = []xml.Attr{}
	for _, attr := range u.Attributes {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: attr.Key}, Value: attr.Value})
	}
	e.EncodeToken(start)
	if len(u.Text) > 0 {
		e.EncodeToken(xml.CharData(u.Text))
	} else {
		for _, elem := range u.Elements {
			e.EncodeElement(elem, xml.StartElement{Name: xml.Name{Local: elem.Name}})
		}
	}
	e.EncodeToken(start.End())
	return nil
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

		var propertyGroupsConfigurations []*MSBuildXMLElement
		var propertyGroupsGenerals []*MSBuildXMLElement
		var itemDefinitionGroups []*MSBuildXMLElement

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

			if len(config.ExecutableExtension) == 0 {
				config.ExecutableExtension = ".exe"
			}
			if len(config.StaticLibraryExtension) == 0 {
				config.StaticLibraryExtension = ".lib"
			}
			if len(config.DynamicLibraryExtension) == 0 {
				config.DynamicLibraryExtension = ".dll"
			}

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
						str += ("$(OutDir)" + dep.Name + config.StaticLibraryExtension)
						str += ";"
					}
				}
				str += "%(AdditionalDependencies)"
				return str
			}()

			conditionStr := fmt.Sprintf("%s|%s", config.Configuration, config.Platform)
			projectSource.Conditions = append(projectSource.Conditions, conditionStr)

			configuration := &MSBuildXMLElement{
				Name: "PropertyGroup",
				Attributes: []MSBuildXMLAttribute{
					xmlAttr("Condition", fmt.Sprintf("'$(Configuration)|$(Platform)'=='%s'", conditionStr)),
					xmlAttr("Label", "Configuration"),
				},
			}
			for k, v := range msbuild.Configuration {
				configuration.SubElement(k).SetText(v)
			}
			propertyGroupsConfigurations = append(propertyGroupsConfigurations, configuration)

			propertyGroupsGeneral := &MSBuildXMLElement{
				Name: "PropertyGroup",
				Attributes: []MSBuildXMLAttribute{
					xmlAttr("Condition", fmt.Sprintf("'$(Configuration)|$(Platform)'=='%s'", conditionStr)),
				},
			}
			for k, v := range msbuild.General {
				configuration.SubElement(k).SetText(v)
			}
			propertyGroupsGenerals = append(propertyGroupsGenerals, propertyGroupsGeneral)

			itemDefinition := &MSBuildXMLElement{
				Name: "ItemDefinitionGroup",
				Attributes: []MSBuildXMLAttribute{
					xmlAttr("Condition", fmt.Sprintf("'$(Configuration)|$(Platform)'=='%s'", conditionStr)),
				},
			}

			if len(msbuild.ClCompile) > 0 {
				item := itemDefinition.SubElement("ClCompile")
				for k, v := range msbuild.ClCompile {
					item.SubElement(k).SetText(v)
				}
			}
			if len(msbuild.Link) > 0 {
				item := itemDefinition.SubElement("Link")
				for k, v := range msbuild.Link {
					item.SubElement(k).SetText(v)
				}
			}
			if len(msbuild.Lib) > 0 {
				item := itemDefinition.SubElement("Lib")
				for k, v := range msbuild.Lib {
					item.SubElement(k).SetText(v)
				}
			}

			itemDefinitionGroups = append(itemDefinitionGroups, itemDefinition)
		}

		projectSource.Project = MSBuildXMLElement{
			Name: "Project",
			Attributes: []MSBuildXMLAttribute{
				xmlAttr("DefaultTargets", "Build"),
				xmlAttr("ToolsVersion", "14.0"),
				xmlAttr("xmlns", "http://schemas.microsoft.com/developer/msbuild/2003"),
			},
		}

		vcxproj := &projectSource.Project

		{
			configurations := vcxproj.SubElement("ItemGroup", xmlAttr("Label", "ProjectConfigurations"))
			for _, v := range project.Configurations {
				include := fmt.Sprintf("%s|%s", v.Configuration, v.Platform)
				conf := configurations.SubElement("ProjectConfiguration", xmlAttr("Include", include))
				conf.SubElement("Configuration").SetText(v.Configuration)
				conf.SubElement("Platform").SetText(v.Platform)
			}
		}
		{
			globals := vcxproj.SubElement("PropertyGroup", xmlAttr("Label", "Globals"))
			globals.SubElement("ProjectGuid").SetText(fmt.Sprintf("{%s}", projectSource.GUID))
			globals.SubElement("Keyword").SetText("Win32Proj")
			globals.SubElement("RootNamespace").SetText(projectSource.Name)
			globals.SubElement("WindowsTargetPlatformVersion").SetText("8.1")
		}

		vcxproj.SubElement("Import", xmlAttr("Project", `$(VCTargetsPath)\Microsoft.Cpp.Default.props`))

		vcxproj.Elements = append(vcxproj.Elements, propertyGroupsConfigurations...)

		vcxproj.SubElement("Import", xmlAttr("Project", `$(VCTargetsPath)\Microsoft.Cpp.props`))
		extensionSettings := vcxproj.SubElement("ImportGroup", xmlAttr("Label", "ExtensionSettings"))
		for _, s := range project.ExtensionSettings {
			extensionSettings.SubElement("Import", xmlAttr("Project", s))
		}

		vcxproj.SubElement("ImportGroup", xmlAttr("Label", "Shared"))

		for _, v := range project.Configurations {
			propertySheets := vcxproj.SubElement(
				"ImportGroup",
				xmlAttr("Label", "PropertySheets"),
				xmlAttr("Condition", fmt.Sprintf("'$(Configuration)|$(Platform)'=='%s|%s'", v.Configuration, v.Platform)),
			)
			propertySheets.SubElement(
				"Import",
				xmlAttr("Project", `$(UserRootDir)\Microsoft.Cpp.$(Platform).user.props`),
				xmlAttr("Condition", `exists('$(UserRootDir)\Microsoft.Cpp.$(Platform).user.props')`),
				xmlAttr("Label", "LocalAppDataPlatform"),
			)
		}

		vcxproj.SubElement("PropertyGroup", xmlAttr("Label", "UserMacros"))
		if len(propertyGroupsGenerals) > 0 {
			vcxproj.Elements = append(vcxproj.Elements, propertyGroupsGenerals...)
		} else {
			vcxproj.SubElement("PropertyGroup")
		}

		vcxproj.Elements = append(vcxproj.Elements, itemDefinitionGroups...)

		clIncludeSources := func() (result []MSBuildXMLItem) {
			for _, src := range edge.GetHeaders(env) {
				src, _ = filepath.Rel(env.OutDir, src)
				result = append(result, MSBuildXMLItem{Include: src})
			}
			return result
		}()
		{
			itemGroup := vcxproj.SubElement("ItemGroup")
			for _, v := range clIncludeSources {
				itemGroup.SubElement("ClInclude", xmlAttr("Include", v.Include))
			}
		}

		clCompileSources := getClCompileSources(edge, &project, env)
		{
			itemGroup := vcxproj.SubElement("ItemGroup")
			for _, v := range clCompileSources {
				item := itemGroup.SubElement("ClCompile", xmlAttr("Include", v.Include))
				for _, e := range v.ExcludedFromBuild {
					item.SubElement("ExcludedFromBuild", xmlAttr("Condition", e.Condition)).SetText(fmt.Sprintf("%v", e.Excluded))
				}
			}
		}

		vcxproj.SubElement("Import", xmlAttr("Project", `$(VCTargetsPath)\Microsoft.Cpp.targets`))
		extensionTargets := vcxproj.SubElement("ImportGroup", xmlAttr("Label", "ExtensionTargets"))
		for _, s := range project.ExtensionTargets {
			extensionTargets.SubElement("Import", xmlAttr("Project", s))
		}

		projectSource.ProjectFilters = MSBuildXMLElement{
			Name: "Project",
			Attributes: []MSBuildXMLAttribute{
				xmlAttr("ToolsVersion", "4.0"),
				xmlAttr("xmlns", "http://schemas.microsoft.com/developer/msbuild/2003"),
			},
		}

		filters := &projectSource.ProjectFilters
		{
			itemGroup := filters.SubElement("ItemGroup")
			{
				s := itemGroup.SubElement("Filter", xmlAttr("Include", "Source Files"))
				s.SubElement("UniqueIdentifier").SetText("{4FC737F1-C7A5-4376-A066-2A32D752A2FF}")
				s.SubElement("Extensions").SetText("cpp;c;cc;cxx;def;odl;idl;hpj;bat;asm;asmx")
			}
			{
				s := itemGroup.SubElement("Filter", xmlAttr("Include", "Header Files"))
				s.SubElement("UniqueIdentifier").SetText("{93995380-89BD-4b04-88EB-625FBE52EBFB}")
				s.SubElement("Extensions").SetText("h;hh;hpp;hxx;hm;inl;inc;xsd")
			}
		}
		{
			itemGroup := filters.SubElement("ItemGroup")
			for _, s := range clIncludeSources {
				s := itemGroup.SubElement("ClInclude", xmlAttr("Include", s.Include))
				s.SubElement("Filter").SetText("Graphics")
			}
		}
		{
			itemGroup := filters.SubElement("ItemGroup")
			for _, s := range clCompileSources {
				s := itemGroup.SubElement("ClCompile", xmlAttr("Include", s.Include))
				s.SubElement("Filter").SetText("Graphics")
			}
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

		const msbuildXMLHeader = `<?xml version="1.0" encoding="utf-8"?>` + "\n"
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
			writer.WriteString(msbuildXMLHeader)
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
			writer.WriteString(msbuildXMLHeader)
			writer.WriteString(replacedXML)

			writer.Flush()
		}

		solution.Projects = append(solution.Projects, projectSource)
	}

	generateMSBuildSolutionFile(env, solution)

	return nil
}
