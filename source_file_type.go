package main

import "path/filepath"

// SourceFileType specifies the file type to associate with the file extension.
type SourceFileType int

const (
	// SourceFileTypeUnknown indicates the file type is unknown.
	SourceFileTypeUnknown SourceFileType = iota

	// SourceFileTypeCppSource indicates the file is a C++ source file.
	SourceFileTypeCppSource

	// SourceFileTypeCppHeader indicates the file is a C++ header file.
	SourceFileTypeCppHeader

	// SourceFileTypeCSource indicates the file is a C source file.
	SourceFileTypeCSource

	// SourceFileTypeObjC indicates the file is a Objective-C source file.
	SourceFileTypeObjC

	// SourceFileTypeObjCpp indicates the file is a Objective-C++ source file.
	SourceFileTypeObjCpp
)

func getSourceFileType(filename string) SourceFileType {
	ext := filepath.Ext(filename)

	if ext == ".cpp" || ext == ".cc" || ext == ".cxx" || ext == ".c++" {
		return SourceFileTypeCppSource
	} else if ext == ".c" {
		return SourceFileTypeCSource
	} else if ext == ".h" || ext == ".hh" || ext == ".hpp" || ext == ".hxx" || ext == ".inc" || ext == ".ipp" {
		return SourceFileTypeCppHeader
	} else if ext == ".m" {
		return SourceFileTypeObjC
	} else if ext == ".mm" {
		return SourceFileTypeObjCpp
	}
	return SourceFileTypeUnknown
}
