package types

import "go.elara.ws/lure/manager"

type BuildOpts struct {
	Script      string
	Manager     manager.Manager
	Clean       bool
	Interactive bool
}

// BuildVars represents the script variables required
// to build a package
type BuildVars struct {
	Name          string   `sh:"name,required"`
	Version       string   `sh:"version,required"`
	Release       int      `sh:"release,required"`
	Epoch         uint     `sh:"epoch"`
	Description   string   `sh:"desc"`
	Homepage      string   `sh:"homepage"`
	Maintainer    string   `sh:"maintainer"`
	Architectures []string `sh:"architectures"`
	Licenses      []string `sh:"license"`
	Provides      []string `sh:"provides"`
	Conflicts     []string `sh:"conflicts"`
	Depends       []string `sh:"deps"`
	BuildDepends  []string `sh:"build_deps"`
	Replaces      []string `sh:"replaces"`
	Sources       []string `sh:"sources"`
	Checksums     []string `sh:"checksums"`
	Backup        []string `sh:"backup"`
	Scripts       Scripts  `sh:"scripts"`
}

type Scripts struct {
	PreInstall  string `sh:"preinstall"`
	PostInstall string `sh:"postinstall"`
	PreRemove   string `sh:"preremove"`
	PostRemove  string `sh:"postremove"`
	PreUpgrade  string `sh:"preupgrade"`
	PostUpgrade string `sh:"postupgrade"`
	PreTrans    string `sh:"pretrans"`
	PostTrans   string `sh:"posttrans"`
}

type Directories struct {
	BaseDir   string
	SrcDir    string
	PkgDir    string
	ScriptDir string
}
