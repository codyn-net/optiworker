package main

var AppConfig = struct {
	// user executables
	BinDir string

	// read-only arc.-independent data
	DataDir string

	// read-only arch.-independent data root
	DataRootDir string

	// install architecture-dependent files in EPREFIX
	ExecPrefix string

	// program executables
	LibDir string

	// program executables
	LibExecDir string

	// man documentation
	ManDir string

	// install architecture-independent files in PREFIX
	Prefix string

	// read-only single-machine data
	SysConfDir string

	// Application version
	Version []int
}{
	"/usr/local/bin",
	"/usr/local/share",
	"/usr/local/share",
	"/usr/local",
	"/usr/local/lib",
	"/usr/local/libexec",
	"/usr/local/share/man",
	"/usr/local",
	"/usr/local/etc",
	[]int{2, 12},
}
