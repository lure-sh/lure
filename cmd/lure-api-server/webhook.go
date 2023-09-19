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
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"

	"go.elara.ws/logger/log"
	"go.elara.ws/lure/internal/config"
	"go.elara.ws/lure/internal/repos"
)

func handleWebhook(sigCh chan<- struct{}) http.HandlerFunc {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if req.Header.Get("X-GitHub-Event") != "push" {
			http.Error(res, "Only push events are accepted by this bot", http.StatusBadRequest)
			return
		}

		err := verifySecure(req)
		if err != nil {
			http.Error(res, err.Error(), http.StatusForbidden)
			return
		}

		sigCh <- struct{}{}
		return
	})
}

func verifySecure(req *http.Request) error {
	sigStr := req.Header.Get("X-Hub-Signature-256")
	sig, err := hex.DecodeString(strings.TrimPrefix(sigStr, "sha256="))
	if err != nil {
		return err
	}

	secretStr, ok := os.LookupEnv("LURE_API_GITHUB_SECRET")
	if !ok {
		return errors.New("LURE_API_GITHUB_SECRET must be set to the secret used for setting up the github webhook\n\n")
	}
	secret := []byte(secretStr)

	h := hmac.New(sha256.New, secret)
	_, err = io.Copy(h, req.Body)
	if err != nil {
		return err
	}

	if !hmac.Equal(h.Sum(nil), sig) {
		log.Warn("Insecure webhook request").
			Str("from", req.RemoteAddr).
			Bytes("sig", sig).
			Bytes("hmac", h.Sum(nil)).
			Send()

		return errors.New("webhook signature mismatch")
	}

	return nil
}

func repoPullWorker(ctx context.Context, sigCh <-chan struct{}) {
	for {
		select {
		case <-sigCh:
			err := repos.Pull(ctx, config.Config().Repos)
			if err != nil {
				log.Warn("Error while pulling repositories").Err(err).Send()
			}
		case <-ctx.Done():
			return
		}
	}
}
