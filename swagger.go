package apiservices

import (
	_ "embed"
	"net/http"
)

//go:embed swagger.html
var swagger []byte

func Swagger(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write(swagger)
}
