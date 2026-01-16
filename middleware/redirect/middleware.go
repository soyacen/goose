package redirect

import (
	"net/http"

	"github.com/soyacen/goose/server"
)

func Server() server.Middleware {
	return func(response http.ResponseWriter, request *http.Request, invoker http.HandlerFunc) {
		// TODO handle the Forwarded-For header when it ratifies
		// http://tools.ietf.org/html/draft-ietf-appsawg-http-forwarded-10
		if request.URL.Scheme != "https" && request.Header.Get("X-Forwarded-Proto") != "https" {
			request.URL.Scheme = "https"
			request.URL.Host = request.Host
			http.Redirect(response, request, request.URL.String(), http.StatusFound)
			return
		}
		invoker(response, request)
	}
}
