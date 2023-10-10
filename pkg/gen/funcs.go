package gen

import "text/template"

var funcs = template.FuncMap{
	"firstchar": func(s string) string {
		return s[:1]
	},
}
