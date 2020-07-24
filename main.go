package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/go-chi/chi"
	"github.com/go-chi/cors"
)

func main() {}

func router() *chi.Mux {
	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		// AllowedOrigins: []string{"https://foo.com"}, // Use this to allow specific origin hosts
		AllowedOrigins: []string{"*"},
		// AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))

	r.Get("/", CfBypass)

	return r
}

// CfBypass is handles your request, changes tls fingerprint, useragent, etc, and bypasses cf
func CfBypass(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	rawurl := query.Get("url")
	redirect, err := url.Parse(rawurl)

	if err != nil || rawurl == "" {
		log.Println("Error ocurred while parsing url ", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	r.Header.Add("User-Agent", "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:78.0) Gecko/20100101 Firefox/78.0")
	proxy := httputil.NewSingleHostReverseProxy(redirect)
	proxy.ServeHTTP(w, r)
}
