package main

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"encoding/base64"
	"encoding/json"

	"github.com/ccaneke/redithistory/cmd/web/response"
	"github.com/google/uuid"
)

type handler struct {
	logger logger
	redis  redisClientInterface
}

var userLogins = map[string]response.Response{}

type logger interface {
	Print(v ...any)
}

type redisClientInterface interface {
}

type httpClient interface {
	Do(*http.Request) (*http.Response, error)
}

const (
	internalServerError = "Internal Server Error"
	methodNotAllowed    = "Method Not Allowed"
	apiAccessDenied     = "access denied"
	stateError          = "state returned by reddit does not match the one sent in the initial authorization request"
)

func (h *handler) home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		h.logger.Print("request URL path does not exactly match /")
		http.NotFound(w, r)
		return
	}

	if r.Method != "GET" {
		w.Header().Set("Allow", http.MethodGet)
		http.NotFound(w, r)
		return
	}

	files := []string{"./ui/html/base.tmpl", "./ui/html/partials/nav.tmpl", "./ui/html/pages/home.tmpl"}

	ts, err := template.ParseFiles(files...)
	if err != nil {
		h.logger.Print(err)
		http.Error(w, internalServerError, http.StatusInternalServerError)
		return
	}

	parameters := getAuthorizationURLParams(r.Host)
	userLogins[parameters.State] = response.Response{}

	err = ts.ExecuteTemplate(w, "base", parameters)
	if err != nil {
		http.Error(w, internalServerError, http.StatusInternalServerError)
		return
	}
}

func (h *handler) search(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, methodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	queryParams := r.URL.Query()
	client := &http.Client{}
	resp, err := h.getBearerToken(queryParams, r, client)
	if err != nil {
		http.Error(w, internalServerError, http.StatusInternalServerError)
		return
	}

	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		h.logger.Print(err)
		http.Error(w, internalServerError, http.StatusInternalServerError)
		return
	}

	tokenResponseObj := response.Response{}

	err = json.Unmarshal(b, &tokenResponseObj)
	if err != nil {
		h.logger.Print(err)
		http.Error(w, internalServerError, http.StatusInternalServerError)
		return
	}

	if _, exists := userLogins[queryParams.Get("state")]; !exists {
		h.logger.Print(errors.New(stateError))
		http.Error(w, internalServerError, http.StatusInternalServerError)
		return
	}

	userLogins[queryParams.Get("state")] = tokenResponseObj

	files := []string{"./ui/html/base.tmpl", "./ui/html/partials/nav.tmpl", "./ui/html/pages/search.tmpl"}

	ts, err := template.ParseFiles(files...)
	if err != nil {
		h.logger.Print(err)
		http.Error(w, internalServerError, http.StatusInternalServerError)
		return
	}

	err = ts.ExecuteTemplate(w, "base", nil)
	if err != nil {
		h.logger.Print(err)
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
		Scope:        "history identity",
	}

	return parameters
}

func (h *handler) getBearerToken(queryParameters url.Values, r *http.Request, client httpClient) (*http.Response, error) {
	code, errMsg := queryParameters.Get("code"), queryParameters.Get("err")
	if errMsg == "access_denied" {
		return nil, errors.New(apiAccessDenied)
	}

	message := url.Values{}
	message.Add("grant_type", "authorization_code")
	message.Add("code", code)
	message.Add("redirect_uri", "http://"+r.Host+r.URL.Path)

	clientID := os.Getenv("ClientID")
	clientSecret := os.Getenv("ClientSecret")

	encoded := base64.StdEncoding.EncodeToString([]byte(clientID + ":" + clientSecret))

	request, err := http.NewRequest("POST", "https://www.reddit.com/api/v1/access_token", strings.NewReader(message.Encode()))
	if err != nil {
		h.logger.Print(err)
		return nil, err
	}

	request.Header.Add("Authorization", "Basic "+encoded)
	request.Header.Add("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/105.0.0.0 Safari/537.36:history:v0.1")
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(request)
	if err != nil {
		h.logger.Print(err)
		return nil, err
	}
	return resp, nil
}
