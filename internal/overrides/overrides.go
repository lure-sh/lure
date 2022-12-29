package overrides

import (
	"runtime"
	"strings"

	"go.arsenm.dev/lure/distro"
	"go.arsenm.dev/lure/internal/cpu"
)

type Opts struct {
	Name        string
	Overrides   bool
	LikeDistros bool
}

var DefaultOpts = &Opts{
	Overrides:   true,
	LikeDistros: true,
}

// Resolve generates a slice of possible override names in the order that they should be checked
func Resolve(info *distro.OSRelease, opts *Opts) []string {
	if opts == nil {
		opts = DefaultOpts
	}

	if !opts.Overrides {
		return []string{opts.Name}
	}

	architectures := []string{runtime.GOARCH}

	if runtime.GOARCH == "arm" {
		// More specific goes first
		architectures[0] = cpu.ARMVariant()
		architectures = append(architectures, "arm")
	}

	distros := []string{info.ID}
	if opts.LikeDistros {
		distros = append(distros, info.Like...)
	}

	var out []string
	for _, arch := range architectures {
		for _, distro := range distros {
			if opts.Name == "" {
				out = append(
					out,
					arch+"_"+distro,
					distro,
				)
			} else {
				out = append(
					out,
					opts.Name+"_"+arch+"_"+distro,
					opts.Name+"_"+distro,
				)
			}
		}
		if opts.Name == "" {
			out = append(out, arch)
		} else {
			out = append(out, opts.Name+"_"+arch)
		}
	}
	if opts.Name != "" {
		out = append(out, opts.Name)
	}

	for index, item := range out {
		out[index] = strings.ReplaceAll(item, "-", "_")
	}

	return out
}

func (o *Opts) WithName(name string) *Opts {
	o.Name = name
	return o
}

func (o *Opts) WithOverrides(v bool) *Opts {
	o.Overrides = v
	return o
}

func (o *Opts) WithLikeDistros(v bool) *Opts {
	o.LikeDistros = v
	return o
}
