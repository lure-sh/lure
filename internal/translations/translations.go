/*
 * LURE - Linux User REpository
 * Copyright (C) 2023 Elara Musayelyan
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

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
