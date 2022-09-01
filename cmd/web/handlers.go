package main

import (
	"fmt"
	"html/template"
	"net/http"
	"os"

	"github.com/google/uuid"
)

type handler struct {
	logger logger
	redis  redisClientInterface
}

type logger interface {
}

type redisClientInterface interface {
}

const (
	NotFound            = "NotFound"
	internalServerError = "Internal Server Error"
)

func (h *handler) home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, NotFound, http.StatusNotFound)
		return
	}

	if r.Method != "GET" {
		http.Error(w, NotFound, http.StatusInternalServerError)
	}

	files := []string{"./ui/html/base.tmpl", "./ui/html/partials/nav.tmpl", "./ui/html/pages/home.tmpl"}

	ts, err := template.ParseFiles(files...)
	if err != nil {
		http.Error(w, internalServerError, http.StatusInternalServerError)
		return
	}

	parameters := getAuthorizationURLParams(r.Host)

	err = ts.ExecuteTemplate(w, "base", parameters)
	if err != nil {
		http.Error(w, internalServerError, http.StatusInternalServerError)
		return
	}
}

func getAuthorizationURLParams(host string) struct {
	ClientID     string
	ResponseType string
	State        string
	RedirectURI  string
	Duration     string
	Scope        string
} {
	parameters := struct {
		ClientID     string
		ResponseType string
		State        string
		RedirectURI  string
		Duration     string
		Scope        string
	}{
		ClientID:     os.Getenv("ClientID"),
		ResponseType: "code",
		State:        uuid.New().String(),
		RedirectURI:  fmt.Sprintf("http://%s/search", host),
		Duration:     "temporary",
		Scope:        "history",
	}

	return parameters
}
