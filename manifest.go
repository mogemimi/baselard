package main

// Templates defines template files for generating project files.
type Templates struct {
	MSBuildProject  string `toml:"vcxproj"`
	MSBuildSolution string `toml:"sln"`
	Ninja           string `toml:"ninja"`
	XcodeProject    string `toml:"xcodeproj"`
}

// Tagged defines tagged configuration settings.
type Tagged struct {
	Headers         []string        `toml:"headers"`
	Sources         []string        `toml:"sources"`
	IncludeDirs     []string        `toml:"include_dirs"`
	LibDirs         []string        `toml:"lib_dirs"`
	Defines         []string        `toml:"defines"`
	Dependencies    []string        `toml:"deps"`
	CompilerFlags   []string        `toml:"cflags"`
	CompilerFlagsC  []string        `toml:"cflags_c"`
	CompilerFlagsCC []string        `toml:"cflags_cc"`
	LinkerFlags     []string        `toml:"ldflags"`
	MSBuildSettings MSBuildSettings `toml:"msbuild_settings"`
	Templates       Templates       `toml:"templates"`
}

// Target defines a build target and configuration settings.
type Target struct {
	Name            string            `toml:"name"`
	Type            string            `toml:"type"`
	Headers         []string          `toml:"headers"`
	Sources         []string          `toml:"sources"`
	IncludeDirs     []string          `toml:"include_dirs"`
	LibDirs         []string          `toml:"lib_dirs"`
	Defines         []string          `toml:"defines"`
	CompilerFlags   []string          `toml:"cflags"`
	CompilerFlagsC  []string          `toml:"cflags_c"`
	CompilerFlagsCC []string          `toml:"cflags_cc"`
	LinkerFlags     []string          `toml:"ldflags"`
	MSBuildSettings MSBuildSettings   `toml:"msbuild_settings"`
	Dependencies    []string          `toml:"deps"`
	Configs         []string          `toml:"configs"`
	Tagged          map[string]Tagged `toml:"tagged"`
	MSBuildProject  MSBuildProject    `toml:"msbuild_project"`
	Templates       Templates         `toml:"templates"`
}

// MSBuildSettings defines configuration settings for MSBuild.
type MSBuildSettings struct {
	ClCompile     map[string]string `toml:"ClCompile"`
	Link          map[string]string `toml:"Link"`
	Lib           map[string]string `toml:"Lib"`
	Globals       map[string]string `toml:"Globals"`
	Configuration map[string]string `toml:"Configuration"`
	User          map[string]string `toml:"User"`
	General       map[string]string `toml:"General"`
}

// MSBuildProjectConfiguration defines configuration settings for MSBuild.
type MSBuildProjectConfiguration struct {
	Platform      string   `toml:"platform"`
	Configuration string   `toml:"configuration"`
	Tags          []string `toml:"tags"`

	// TODO: Move the following definitions to out of MSBuildProjectConfiguration
	ExecutableExtension     string `toml:"executable_extension"`
	StaticLibraryExtension  string `toml:"static_library_extension"`
	DynamicLibraryExtension string `toml:"dynamic_library_extension"`
}

// MSBuildProject defines configurations and platforms for MSBuild.
type MSBuildProject struct {
	Configurations    []MSBuildProjectConfiguration `toml:"configurations"`
	ExtensionSettings []string                      `toml:"ExtensionSettings"`
	ExtensionTargets  []string                      `toml:"ExtensionTargets"`
}

// Manifest represents a input build settings.
type Manifest struct {
	Targets []Target `toml:"targets"`
}
