package swagger

import (
	_ "embed"
	"net/http"
)

//go:embed swagger.html
var swagger []byte

func Home(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write(swagger)
}
