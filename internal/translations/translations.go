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
	"context"
	"embed"
	"sync"

	"go.elara.ws/logger"
	"lure.sh/lure/pkg/loggerctx"
	"go.elara.ws/translate"
	"golang.org/x/text/language"
)

//go:embed files
var translationFS embed.FS

var (
	mu         sync.Mutex
	translator *translate.Translator
)

func Translator(ctx context.Context) *translate.Translator {
	mu.Lock()
	defer mu.Unlock()
	log := loggerctx.From(ctx)
	if translator == nil {
		t, err := translate.NewFromFS(translationFS)
		if err != nil {
			log.Fatal("Error creating new translator").Err(err).Send()
		}
		translator = &t
	}
	return translator
}

func NewLogger(ctx context.Context, l logger.Logger, lang language.Tag) *translate.TranslatedLogger {
	return translate.NewLogger(l, *Translator(ctx), lang)
}
