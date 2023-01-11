package overrides

import (
	"os"
	"runtime"
	"strings"

	"go.arsenm.dev/lure/distro"
	"go.arsenm.dev/lure/internal/cpu"
	"golang.org/x/exp/slices"
	"golang.org/x/text/language"
)

type Opts struct {
	Name        string
	Overrides   bool
	LikeDistros bool
	Languages   []string
}

var DefaultOpts = &Opts{
	Overrides:   true,
	LikeDistros: true,
	Languages:   []string{"en"},
}

// Resolve generates a slice of possible override names in the order that they should be checked
func Resolve(info *distro.OSRelease, opts *Opts) ([]string, error) {
	if opts == nil {
		opts = DefaultOpts
	}

	if !opts.Overrides {
		return []string{opts.Name}, nil
	}

	langs, err := parseLangs(opts.Languages)
	if err != nil {
		return nil, err
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
	out = append(out, opts.Name)

	for index, item := range out {
		out[index] = strings.ReplaceAll(item, "-", "_")
	}

	if len(langs) > 0 {
		tmp := out
		out = make([]string, 0, len(tmp)+(len(tmp)*len(langs)))
		for _, lang := range langs {
			for _, val := range tmp {
				if val == "" {
					continue
				}

				out = append(out, val+"_"+lang)
			}
		}
		out = append(out, tmp...)
	}

	return out, nil
}

func (o *Opts) WithName(name string) *Opts {
	out := &Opts{}
	*out = *o

	out.Name = name
	return out
}

func (o *Opts) WithOverrides(v bool) *Opts {
	out := &Opts{}
	*out = *o

	out.Overrides = v
	return out
}

func (o *Opts) WithLikeDistros(v bool) *Opts {
	out := &Opts{}
	*out = *o

	out.LikeDistros = v
	return out
}

func parseLangs(langs []string) ([]string, error) {
	out := make([]string, len(langs))
	for i, lang := range langs {
		tag, err := language.Parse(lang)
		if err != nil {
			return nil, err
		}
		base, _ := tag.Base()
		out[i] = base.String()
	}
	slices.Sort(out)
	out = slices.Compact(out)
	return out, nil
}

func SystemLang() string {
	lang := os.Getenv("LANG")
	if lang == "" {
		lang = "en"
	}
	return lang
}