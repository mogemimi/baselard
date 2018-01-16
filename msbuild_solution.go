package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
)

// MSBuildSolution reperesents a solution file in Visual Studio.
type MSBuildSolution struct {
	Name     string
	Projects []*MSBuildProjectFile
}

func removeDuplicatesFromSlice(in []string) []string {
	results := make([]string, 0, len(in))
	encountered := map[string]bool{}
	for i := 0; i < len(in); i++ {
		if !encountered[in[i]] {
			encountered[in[i]] = true
			results = append(results, in[i])
		}
	}
	return results
}

func generateMSBuildSolutionFile(solutionFilePath string, solution *MSBuildSolution) (err error) {
	str := `Microsoft Visual Studio Solution File, Format Version 12.00
# Visual Studio 14
VisualStudioVersion = 14.0.24720.0
MinimumVisualStudioVersion = 10.0.40219.1
`

	// NOTE:
	// The following text value is a GUID that specifies a Visual C++ project.
	// Please see also https://msdn.microsoft.com/en-us/library/hb23x61k(v=vs.80).aspx
	const projectTypeGUID string = "{8BC9CEB8-8B4A-11D0-8D11-00A0C91BC942}"

	for _, proj := range solution.Projects {
		path, err := filepath.Rel(filepath.Dir(solutionFilePath), proj.FilePath)
		if err != nil {
			path = proj.FilePath
		}
		str += fmt.Sprintf("Project(\"%s\") = \"%s\", \"%s\", \"%s\"\n", projectTypeGUID, proj.Name, path, proj.GUID)
		if len(proj.DependProjects) > 0 {
			str += "	ProjectSection(ProjectDependencies) = postProject\n"
			for _, depend := range proj.DependProjects {
				str += fmt.Sprintf("		{%s} = {%s}\n", depend, depend)
			}
			str += "	EndProjectSection\n"
		}
		str += "EndProject\n"
	}

	str += "Global\n"
	str += "\tGlobalSection(SolutionConfigurationPlatforms) = preSolution\n"
	str += func() (out string) {
		conditions := []string{}
		for _, proj := range solution.Projects {
			conditions = append(conditions, proj.Conditions...)
		}
		conditions = removeDuplicatesFromSlice(conditions)
		sort.Slice(conditions, func(i, j int) bool {
			return conditions[i] < conditions[j]
		})

		for _, cond := range conditions {
			out += ("\t\t" + cond + " = " + cond + "\n")
		}
		return out
	}()
	str += "\tEndGlobalSection\n"

	str += "\tGlobalSection(ProjectConfigurationPlatforms) = postSolution\n"
	str += func() (out string) {
		for _, proj := range solution.Projects {
			for _, cond := range proj.Conditions {
				out += fmt.Sprintf("\t\t%s.%s.ActiveCfg = %s\n", proj.GUID, cond, cond)
				out += fmt.Sprintf("\t\t%s.%s.Build.0 = %s\n", proj.GUID, cond, cond)
			}
		}
		return out
	}()
	str += "\tEndGlobalSection\n"

	str += `	GlobalSection(SolutionProperties) = preSolution
		HideSolutionNode = FALSE
	EndGlobalSection
`
	str += "EndGlobal\n"

	content := []byte(str)
	err = ioutil.WriteFile(solutionFilePath, content, os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}
