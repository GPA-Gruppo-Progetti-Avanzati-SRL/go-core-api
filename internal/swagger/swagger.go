package swagger

import (
	_ "embed"
	"net/http"

	"github.com/rs/zerolog/log"
)

//go:embed swagger.html
var swagger []byte

func Home(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	// Lo status è già partito con il primo Write: qui non c'è più un errore da riportare al
	// client, resta solo da scriverlo nel log (tipicamente un client che ha chiuso la connessione).
	if _, err := w.Write(swagger); err != nil {
		log.Warn().Err(err).Msg("invio della pagina Swagger non completato")
	}
}
