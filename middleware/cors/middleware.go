package cors

import (
	"net/http"
	"strconv"
	"time"

	"github.com/soyacen/goose/server"
)

type options struct {
	AllowOrigin func(*http.Request) string
	MaxAge      time.Duration
}

type Option func(*options)

func AllowOrigin(origin func(*http.Request) string) Option {
	return func(cfg *options) {
		cfg.AllowOrigin = origin
	}
}

func Server(opts ...Option) server.Middleware {
	opt := &options{
		AllowOrigin: func(request *http.Request) string {
			return "*"
		},
		MaxAge: 10 * time.Minute,
	}
	age := strconv.Itoa(int(opt.MaxAge / time.Second))
	return func(response http.ResponseWriter, request *http.Request, invoker http.HandlerFunc) {
		response.Header().Set("Access-Control-Allow-Methods", "GET")
		response.Header().Set("Access-Control-Allow-Headers", "Accept, Accept-Encoding, Authorization, Content-Type, Origin")
		response.Header().Set("Access-Control-Allow-Origin", opt.AllowOrigin(request))
		switch request.Method {
		default:
			response.WriteHeader(http.StatusMethodNotAllowed)
		case "OPTIONS":
			if request.Header.Get("Access-Control-Request-Method") == "GET" {
				response.Header().Set("Access-Control-Max-Age", age)
				return
			}
			response.WriteHeader(http.StatusUnauthorized)
		case "HEAD", "GET":
			invoker(response, request)
		}
	}
}
