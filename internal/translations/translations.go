package translations

import (
	"embed"

	"go.elara.ws/logger"
	"go.elara.ws/lure/internal/log"
	"go.elara.ws/translate"
	"golang.org/x/text/language"
)

//go:embed files
var translationFS embed.FS

var translator *translate.Translator

func Translator() *translate.Translator {
	if translator == nil {
		t, err := translate.NewFromFS(translationFS)
		if err != nil {
			log.Fatal("Error creating new translator").Err(err).Send()
		}
		translator = &t
	}
	return translator
}

func NewLogger(l logger.Logger, lang language.Tag) *translate.TranslatedLogger {
	return translate.NewLogger(l, *Translator(), lang)
}
