package middleware

import (
	"net/http"
)

type RouteFunc func(*http.Request) string
