package main

import (
	"encoding/xml"
	"testing"
)

func TestToMSBuildXMLElement(t *testing.T) {
	vcxproj := &MSBuildXMLElement{
		Name: "Project",
		Attributes: []MSBuildXMLAttribute{
			xmlAttr("DefaultTargets", "Build"),
			xmlAttr("ToolsVersion", "14.0"),
			xmlAttr("xmlns", "http://schemas.microsoft.com/developer/msbuild/2003"),
		},
	}

	xmlString, err := xml.MarshalIndent(vcxproj, "", "  ")
	if err != nil {
		t.Errorf("XML marshal error: %v", err)
	}

	actual := string(xmlString)
	expected := `<Project DefaultTargets="Build" ToolsVersion="14.0" xmlns="http://schemas.microsoft.com/developer/msbuild/2003"></Project>`
	if actual != expected {
		t.Errorf("Unexpected string:\n%v", actual)
	}
}
