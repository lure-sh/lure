package config

import (
	"os"
	"strings"

	"go.arsenm.dev/logger/log"
	"golang.org/x/text/language"
)

var Language language.Tag

func init() {
	lang := SystemLang()
	tag, err := language.Parse(lang)
	if err != nil {
		log.Fatal("Error parsing system language").Err(err).Send()
	}
	base, _ := tag.Base()
	Language = language.Make(base.String())
}

func SystemLang() string {
	lang := os.Getenv("LANG")
	lang, _, _ = strings.Cut(lang, ".")
	if lang == "" {
		lang = "en"
	}
	return lang
}
