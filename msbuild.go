package main

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
)

// MSBuildGenerator generates project files and solution files for Visual Studio.
type MSBuildGenerator struct {
	Projects []*MSBuildProjectFile
}

// MSBuildProjectFile represents a project file for Visual Studio.
type MSBuildProjectFile struct {
	Name           string
	GUID           string
	FilePath       string
	Conditions     []string
	DependProjects []string
	Project        *MSBuildXMLElement
	ProjectFilters *MSBuildXMLElement
}

// MSBuildXMLExcludedFromBuild represents a XML element used in *.vcxproj.
type MSBuildXMLExcludedFromBuild struct {
	Condition string
	Excluded  bool
}

// MSBuildXMLItem represents a XML element used in *.vcxproj.
type MSBuildXMLItem struct {
	Include           string
	ExcludedFromBuild []MSBuildXMLExcludedFromBuild
	Filter            string
}

func getClCompileSources(node *Node, project *MSBuildProject, env *Environment) (result []MSBuildXMLItem) {
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

		for _, src := range node.GetSources(projectEnv) {
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

func sortSubElements(elements []*MSBuildXMLElement) {
	sort.Slice(elements, func(i, j int) bool {
		return elements[i].Name < elements[j].Name
	})
}

// Generate generates projects from a project dependency graph.
func (generator *MSBuildGenerator) Generate(env *Environment, graph *Graph) {
	projectSourceMap := map[*Node]*MSBuildProjectFile{}

	for _, node := range graph.Nodes {
		if node.Type == OutputTypeUnknown {
			continue
		}

		project := &MSBuildProjectFile{
			Name:     node.Name,
			FilePath: filepath.Join(env.ProjectFileDir, node.Name+".vcxproj"),
		}

		guid := uuid.NewV5(uuid.NamespaceDNS, project.FilePath)
		project.GUID = strings.ToUpper(guid.String())

		projectSourceMap[node] = project
		generator.Projects = append(generator.Projects, project)
	}

	for _, node := range graph.Nodes {
		project := projectSourceMap[node]

		for _, dep := range node.Dependencies {
			if depProject, ok := projectSourceMap[dep]; ok {
				project.DependProjects = append(project.DependProjects, depProject.GUID)
			}
		}
	}

	for _, node := range graph.Nodes {
		if node.Type == OutputTypeUnknown {
			continue
		}

		projectSource := projectSourceMap[node]

		project := node.GetMSBuildProject(env)

		var propertyGroupsConfigurations []*MSBuildXMLElement
		var propertyGroupsGenerals []*MSBuildXMLElement
		var itemDefinitionGroups []*MSBuildXMLElement

		configurationType := func() string {
			switch node.Type {
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

			msbuild := node.GetMSBuildSettings(projectEnv)

			msbuild.Configuration["ConfigurationType"] = configurationType

			msbuild.ClCompile["AdditionalIncludeDirectories"] = func() string {
				str := ""
				for _, dir := range node.GetIncludeDirs(projectEnv) {
					dir, _ = filepath.Rel(env.OutDir, dir)
					str += dir
					str += ";"
				}
				str += "%(AdditionalIncludeDirectories)"
				return str
			}()

			msbuild.ClCompile["PreprocessorDefinitions"] = func() string {
				str := ""
				for _, def := range node.GetDefines(projectEnv) {
					str += def
					str += ";"
				}
				str += "%(PreprocessorDefinitions)"
				return str
			}()

			msbuildLinker := func() map[string]string {
				switch node.Type {
				case OutputTypeExecutable:
					return msbuild.Link
				case OutputTypeStaticLibrary:
					return msbuild.Lib
				}
				return msbuild.Link
			}()

			msbuildLinker["AdditionalLibraryDirectories"] = func() string {
				str := ""
				for _, dir := range node.GetLibDirs(projectEnv) {
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
				for _, dep := range node.Dependencies {
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
			sortSubElements(configuration.Elements)
			propertyGroupsConfigurations = append(propertyGroupsConfigurations, configuration)

			propertyGroupsGeneral := &MSBuildXMLElement{
				Name: "PropertyGroup",
				Attributes: []MSBuildXMLAttribute{
					xmlAttr("Condition", fmt.Sprintf("'$(Configuration)|$(Platform)'=='%s'", conditionStr)),
				},
			}
			for k, v := range msbuild.General {
				propertyGroupsGeneral.SubElement(k).SetText(v)
			}
			sortSubElements(propertyGroupsGeneral.Elements)
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
				sortSubElements(item.Elements)
			}
			if len(msbuild.Link) > 0 {
				item := itemDefinition.SubElement("Link")
				for k, v := range msbuild.Link {
					item.SubElement(k).SetText(v)
				}
				sortSubElements(item.Elements)
			}
			if len(msbuild.Lib) > 0 {
				item := itemDefinition.SubElement("Lib")
				for k, v := range msbuild.Lib {
					item.SubElement(k).SetText(v)
				}
				sortSubElements(item.Elements)
			}

			itemDefinitionGroups = append(itemDefinitionGroups, itemDefinition)
		}

		sort.Slice(projectSource.Conditions, func(i, j int) bool {
			return projectSource.Conditions[i] < projectSource.Conditions[j]
		})

		projectSource.Project = &MSBuildXMLElement{
			Name: "Project",
			Attributes: []MSBuildXMLAttribute{
				xmlAttr("DefaultTargets", "Build"),
				xmlAttr("ToolsVersion", "14.0"),
				xmlAttr("xmlns", "http://schemas.microsoft.com/developer/msbuild/2003"),
			},
		}
		vcxproj := projectSource.Project

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
			for _, src := range node.GetHeaders(env) {
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

		clCompileSources := getClCompileSources(node, &project, env)
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

		projectSource.ProjectFilters = &MSBuildXMLElement{
			Name: "Project",
			Attributes: []MSBuildXMLAttribute{
				xmlAttr("ToolsVersion", "4.0"),
				xmlAttr("xmlns", "http://schemas.microsoft.com/developer/msbuild/2003"),
			},
		}
		filters := projectSource.ProjectFilters

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

const (
	msbuildXMLHeader = `<?xml version="1.0" encoding="utf-8"?>` + "\n"
)

// WriteFile writes *.vcxproj formatted xml to a file named by filename.
func (project *MSBuildProjectFile) WriteFile(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	xmlString, err := xml.MarshalIndent(project.Project, "", "  ")
	if err != nil {
		return err
	}

	// TODO: The following solution is too bad.
	replacedXML := strings.Replace(string(xmlString), "&#39;", "'", -1)

	writer := bufio.NewWriter(file)
	writer.WriteString(msbuildXMLHeader)
	writer.WriteString(replacedXML)

	writer.Flush()
	return nil
}

// WriteFiltersFile writes *.vcxproj.filters formatted xml to a file named by filename.
func (project *MSBuildProjectFile) WriteFiltersFile(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	xmlString, err := xml.MarshalIndent(project.ProjectFilters, "", "  ")
	if err != nil {
		return err
	}

	// TODO: The following solution is too bad.
	replacedXML := strings.Replace(string(xmlString), "&#39;", "'", -1)

	writer := bufio.NewWriter(file)
	writer.WriteString(msbuildXMLHeader)
	writer.WriteString(replacedXML)

	writer.Flush()
	return nil
}

// WriteFile writes all files needed for the MSBuild, including *.sln, *.vcxproj and *.vcxproj.filters.
func (generator *MSBuildGenerator) WriteFile(env *Environment) error {
	for _, project := range generator.Projects {
		dir := filepath.Dir(project.FilePath)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err := os.MkdirAll(dir, os.ModePerm); err != nil {
				return errors.Wrapf(err, "Failed to create output directory \"%s\"", dir)
			}
		}

		project.WriteFile(project.FilePath)
		project.WriteFiltersFile(project.FilePath + ".filters")
	}

	solution := &MSBuildSolution{
		Name:     "out",
		Projects: generator.Projects,
	}

	solutionFilePath := filepath.Join(env.OutDir, solution.Name+".sln")
	generateMSBuildSolutionFile(solutionFilePath, solution)

	fmt.Println("Generate project files:")
	fmt.Println(" ", solutionFilePath)
	for _, projectSource := range generator.Projects {
		fmt.Println(" ", projectSource.FilePath)
	}

	return nil
}
