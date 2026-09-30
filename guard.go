package coreapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/rs/zerolog/log"
)

// recoverer trasforma il panic di un handler in un 500 in forma DefaultError, loggato con lo stack.
// Senza, net/http recupera da sé ma chiude la connessione: il client vede una risposta vuota, e nei
// log non resta che una riga di net/http senza la richiesta che l'ha causata — nessun 500 nelle
// metriche, nessun modo di collegare il crash alla chiamata.
//
// http.ErrAbortHandler è il panic con cui un handler (o il reverse proxy) interrompe apposta la
// risposta: si rilancia, come fa net/http.
func recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			p := recover()
			if p == nil {
				return
			}
			if p == http.ErrAbortHandler {
				panic(p)
			}
			log.Error().Str("method", r.Method).Str("path", r.URL.Path).
				Str("panic", fmt.Sprint(p)).Bytes("stack", debug.Stack()).
				Msg("panic in un handler HTTP")
			writeDefaultError(w, http.StatusInternalServerError, CodePanic, "errore interno")
		}()
		next.ServeHTTP(w, r)
	})
}

// limitBody avvolge il body di ogni richiesta in un http.MaxBytesReader: chi legge oltre il limite
// riceve un *http.MaxBytesError (e il server chiude la connessione dopo la risposta). Il
// ValidatorHandler lo traduce in un 413. max <= 0: nessun limite.
func limitBody(max int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if max <= 0 {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil && r.Body != http.NoBody {
				r.Body = http.MaxBytesReader(w, r.Body, max)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// bodyReadError dà status e codice di un errore di lettura del body.
func bodyReadError(err error) (int, string, string) {
	if mbe, ok := errors.AsType[*http.MaxBytesError](err); ok {
		return http.StatusRequestEntityTooLarge, CodeBodyTooLarge, fmt.Sprintf("body oltre il limite di %d byte", mbe.Limit)
	}
	return http.StatusBadRequest, CodeBodyRead, "body della richiesta non leggibile"
}

func writeDefaultError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(&DefaultError{Ambit: Ambit, Code: code, Message: msg}); err != nil {
		log.Warn().Err(err).Str("code", code).Msg("invio della risposta d'errore non completato")
	}
}
