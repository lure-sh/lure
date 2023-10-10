package gen

import (
	"strings"
	"text/template"
)

var funcs = template.FuncMap{
	"tolower": strings.ToLower,
	"firstchar": func(s string) string {
		return s[:1]
	},
}
