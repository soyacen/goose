package basicauth

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"github.com/soyacen/goose/client"
	"github.com/soyacen/goose/server"
)

type ctxKey struct{}

func FromContext(ctx context.Context) (string, bool) {
	value, ok := ctx.Value(ctxKey{}).(string)
	return value, ok
}

type Account struct {
	User     string
	Password string
}

type Accounts []Account

type options struct {
	realm string
}

func (o *options) apply(opts ...Option) *options {
	for _, opt := range opts {
		opt(o)
	}
	return o
}

type Option func(o *options)

func defaultOptions() *options {
	return &options{
		realm: "Authorization Required",
	}
}

func Realm(realm string) Option {
	return func(o *options) {
		o.realm = realm
	}
}

func Server(accounts Accounts, opts ...Option) server.Middleware {
	opt := defaultOptions().apply(opts...)
	realm := "Basic realm=" + strconv.Quote(opt.realm)
	pairs := processAccounts(accounts)
	return func(response http.ResponseWriter, request *http.Request, invoker http.HandlerFunc) {
		user, found := pairs.searchCredential(request.Header.Get("Authorization"))
		if !found {
			response.Header().Set("WWW-Authenticate", realm)
			response.WriteHeader(http.StatusUnauthorized)
			return
		}
		request = request.WithContext(context.WithValue(request.Context(), ctxKey{}, user))
		invoker(response, request)
	}
}

func Client(account Account) client.Middleware {
	return func(cli *http.Client, request *http.Request, invoker client.Invoker) (*http.Response, error) {
		request.URL.User = url.UserPassword(account.User, account.Password)
		return invoker(cli, request)
	}
}

func processAccounts(accounts Accounts) authPairs {
	length := len(accounts)
	if length <= 0 {
		panic(errors.New("basicauth: empty list of authorized credentials"))
	}
	pairs := make(authPairs, 0, length)
	for _, account := range accounts {
		if account.User == "" {
			panic(errors.New("basicauth: user can not be empty"))
		}
		base := account.User + ":" + account.Password
		value := "Basic " + base64.StdEncoding.EncodeToString([]byte(base))
		pairs = append(pairs, authPair{value: value, user: account.User})
	}
	return pairs
}

type authPair struct {
	value string
	user  string
}

type authPairs []authPair

func (a authPairs) searchCredential(authValue string) (string, bool) {
	if authValue == "" {
		return "", false
	}
	for _, pair := range a {
		if subtle.ConstantTimeCompare([]byte(pair.value), []byte(authValue)) == 1 {
			return pair.user, true
		}
	}
	return "", false
}
