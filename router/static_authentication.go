package router

import (
	"log"
	"net/http"
	"net/url"

	"github.com/gorilla/securecookie"
)

type staticAuthentication struct {
	LoginPageHandlerURL string            `json:"login_page"`
	SuccessPath         string            `json:"success"`
	Credentials         map[string]string `json:"credentials"`
	CookieSecret        string            `json:"cookie_secret"`
	CookieName          string            `json:"cookie_name"`
	securecookie        *securecookie.SecureCookie
	r                   *Router
	loginURL            *url.URL
}

func (a *staticAuthentication) finalize() error {
	var err error

	a.loginURL, err = parseHandlerURL(a.LoginPageHandlerURL)
	if err != nil {
		return err
	}

	a.securecookie = securecookie.New([]byte(a.CookieSecret), nil)

	return nil
}

func (a *staticAuthentication) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/logout" {
		a.deleteCookie(w)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if r.URL.Path != "/login" {
		http.NotFound(w, r)
		return
	}

	if r.Method == http.MethodGet {
		*r.URL = *a.loginURL
		a.r.routeInternal(w, r)
		return
	}

	if r.Method == http.MethodPost {
		log.Println("post")
		if err := r.ParseForm(); err != nil {
			internalServerError(w)
			return
		}

		user := r.Form.Get("u")
		pass := r.Form.Get("p")

		log.Println(user, pass)
		if a.tryLogin(user, pass) == true {
			a.setCookie(w)
			http.Redirect(w, r, a.SuccessPath, http.StatusSeeOther)
			return
		}

		w.WriteHeader(http.StatusForbidden)
		return
	}

	w.WriteHeader(http.StatusMethodNotAllowed)
}

//TODO: subject to timing attacks
func (a *staticAuthentication) tryLogin(user string, pass string) bool {
	userPass, hasUser := a.Credentials[user]
	return hasUser && userPass == pass
}

func (a *staticAuthentication) authenticate(r *http.Request) bool {
	cookie, err := r.Cookie(a.CookieName)
	if err != nil {
		return false
	}

	var value bool
	if err := a.securecookie.Decode(a.CookieName, cookie.Value, &value); err != nil {
		return false
	}

	return value
}

func (a *staticAuthentication) setCookie(w http.ResponseWriter) error {
	encoded, err := a.securecookie.Encode(a.CookieName, true)
	if err != nil {
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name:  a.CookieName,
		Value: encoded,
		Path:  "/",
	})

	return nil
}

func (a *staticAuthentication) deleteCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:   a.CookieName,
		MaxAge: -1,
	})
}