package main

import (
	_ "embed"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.elara.ws/lure/pkg/db"
)

//go:embed badge-logo.txt
var logoData string

var _ http.HandlerFunc

func handleBadge() http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		repo := chi.URLParam(req, "repo")
		name := chi.URLParam(req, "pkg")

		pkg, err := db.GetPkg("name = ? AND repository = ?", name, repo)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(res, req, genBadgeURL(pkg.Name, genVersion(pkg)), http.StatusFound)
	}
}

func genVersion(pkg *db.Package) string {
	sb := strings.Builder{}
	if pkg.Epoch != 0 {
		sb.WriteString(strconv.Itoa(int(pkg.Epoch)))
		sb.WriteByte(':')
	}

	sb.WriteString(pkg.Version)

	if pkg.Release != 0 {
		sb.WriteByte('-')
		sb.WriteString(strconv.Itoa(pkg.Release))
	}
	return sb.String()
}

func genBadgeURL(pkgName, pkgVersion string) string {
	v := url.Values{}
	v.Set("label", pkgName)
	v.Set("message", pkgVersion)
	v.Set("logo", logoData)
	v.Set("color", "blue")
	u := &url.URL{Scheme: "https", Host: "img.shields.io", Path: "/static/v1", RawQuery: v.Encode()}
	return u.String()
}
