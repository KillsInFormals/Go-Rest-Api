package middlewares

import (
	"fmt"
	"net/http"
	"strings"
)

type HPPOptions struct {
	CheckQuery                  bool
	CheckBody                   bool
	CheckBodyOnlyForContentType string
	Whitelist                   []string
}

func HPP(options HPPOptions) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if options.CheckBody && r.Method == http.MethodPost && isCorrectContentType(r, options.CheckBodyOnlyForContentType) {
				//filter body params
				filterBodyParams(r, options.Whitelist)
			}

			if options.CheckQuery && r.URL.Query() != nil {
				//filter query params
				filterQueryParams(r, options.Whitelist)
			}
			next.ServeHTTP(w, r)
		})
	}
}

func isCorrectContentType(r *http.Request, contentType string) bool {
	return strings.Contains(r.Header.Get("Content-Type"), contentType)

}

func filterBodyParams(r *http.Request, Whitelist []string) {
	err := r.ParseForm()
	if err != nil {
		fmt.Println(err)
		return
	}

	for k, v := range r.Form {
		if len(v) > 0 {
			r.Form.Set(k, v[0])
		}

		if !isWhitelisted(k, Whitelist) {
			delete(r.Form, k)
		}
	}

}

func filterQueryParams(r *http.Request, Whitelist []string) {

	query := r.URL.Query()

	for k, v := range query {
		if len(v) > 0 {
			query.Set(k, v[0])
		}

		if !isWhitelisted(k, Whitelist) {
			query.Del(k)
		}
	}
	r.URL.RawQuery = query.Encode()
}

func isWhitelisted(param string, whitelist []string) bool {
	for _, v := range whitelist {
		if v == param {
			return true
		}
	}
	return false
}
