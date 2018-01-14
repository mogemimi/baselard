package main

import "encoding/xml"

// MSBuildXMLAttribute represents a xml attribute for Visual Studio projects.
type MSBuildXMLAttribute struct {
	Key   string
	Value string
}

func xmlAttr(key, value string) MSBuildXMLAttribute {
	return MSBuildXMLAttribute{Key: key, Value: value}
}

// MSBuildXMLElement represents a xml element for Visual Studio projects.
type MSBuildXMLElement struct {
	Name       string
	Text       string
	Attributes []MSBuildXMLAttribute
	Elements   []*MSBuildXMLElement
}

// SubElement adds a new element to the sub elements.
func (elem *MSBuildXMLElement) SubElement(name string, attributes ...MSBuildXMLAttribute) *MSBuildXMLElement {
	sub := &MSBuildXMLElement{
		Name:       name,
		Attributes: attributes,
	}
	elem.Elements = append(elem.Elements, sub)
	return sub
}

// SetText sets the text of a element.
func (elem *MSBuildXMLElement) SetText(text string) *MSBuildXMLElement {
	elem.Text = text
	return elem
}

// MarshalXML encodes the receiver as a XML element.
func (elem *MSBuildXMLElement) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = elem.Name
	start.Attr = []xml.Attr{}
	for _, attr := range elem.Attributes {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: attr.Key}, Value: attr.Value})
	}
	e.EncodeToken(start)
	if len(elem.Text) > 0 {
		e.EncodeToken(xml.CharData(elem.Text))
	} else {
		for _, sub := range elem.Elements {
			e.EncodeElement(sub, xml.StartElement{Name: xml.Name{Local: sub.Name}})
		}
	}
	e.EncodeToken(start.End())
	return nil
}
