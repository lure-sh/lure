/*
 * LURE - Linux User REpository
 * Copyright (C) 2023 Arsen Musayelyan
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

package main

import (
	"context"
	"flag"
	"net"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/twitchtv/twirp"
	"go.elara.ws/logger"
	"go.elara.ws/logger/log"
	"go.elara.ws/lure/internal/api"
	"go.elara.ws/lure/internal/config"
	"go.elara.ws/lure/internal/repos"
)

func init() {
	log.Logger = logger.NewPretty(os.Stderr)
}

func main() {
	ctx := context.Background()

	addr := flag.String("a", ":8080", "Listen address for API server")
	logFile := flag.String("l", "", "Output file for JSON log")
	flag.Parse()

	if *logFile != "" {
		fl, err := os.Create(*logFile)
		if err != nil {
			log.Fatal("Error creating log file").Err(err).Send()
		}
		defer fl.Close()

		log.Logger = logger.NewMulti(log.Logger, logger.NewJSON(fl))
	}

	err := repos.Pull(ctx, config.Config().Repos)
	if err != nil {
		log.Fatal("Error pulling repositories").Err(err).Send()
	}

	sigCh := make(chan struct{}, 200)
	go repoPullWorker(ctx, sigCh)

	apiServer := api.NewAPIServer(
		lureWebAPI{},
		twirp.WithServerPathPrefix(""),
	)

	r := chi.NewRouter()
	r.With(allowAllCORSHandler, withAcceptLanguage).Handle("/*", apiServer)
	r.Post("/webhook", handleWebhook(sigCh))
	r.Get("/badge/{repo}/{pkg}", handleBadge())

	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatal("Error starting listener").Err(err).Send()
	}

	log.Info("Starting HTTP API server").Str("addr", ln.Addr().String()).Send()

	err = http.Serve(ln, r)
	if err != nil {
		log.Fatal("Error while running server").Err(err).Send()
	}
}

func allowAllCORSHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.Header().Set("Access-Control-Allow-Origin", "*")
		res.Header().Set("Access-Control-Allow-Headers", "*")
		if req.Method == http.MethodOptions {
			return
		}
		h.ServeHTTP(res, req)
	})
}

type (
	acceptLanguageKey struct{}
	langParameterKey  struct{}
)

func withAcceptLanguage(h http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		ctx := req.Context()

		langs := req.Header.Get("Accept-Language")
		ctx = context.WithValue(ctx, acceptLanguageKey{}, langs)

		lang := req.URL.Query().Get("lang")
		ctx = context.WithValue(ctx, langParameterKey{}, lang)

		req = req.WithContext(ctx)

		h.ServeHTTP(res, req)
	})
}
