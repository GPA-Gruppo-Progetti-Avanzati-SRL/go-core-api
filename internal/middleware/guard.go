package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"

	apierrors "github.com/GPA-Gruppo-Progetti-Avanzati-SRL/go-core-api/internal/errors"
	"github.com/rs/zerolog/log"
)

// Recoverer trasforma il panic di un handler in un 500 in forma DefaultError, loggato con lo stack.
// Senza, net/http recupera da sé ma chiude la connessione: il client vede una risposta vuota, e nei
// log non resta che una riga di net/http senza la richiesta che l'ha causata — nessun 500 nelle
// metriche, nessun modo di collegare il crash alla chiamata.
//
// http.ErrAbortHandler è il panic con cui un handler (o il reverse proxy) interrompe apposta la
// risposta: si rilancia, come fa net/http.
func Recoverer(next http.Handler) http.Handler {
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
			apierrors.Write(w, http.StatusInternalServerError, apierrors.CodePanic, "errore interno")
		}()
		next.ServeHTTP(w, r)
	})
}

// LimitBody avvolge il body di ogni richiesta in un http.MaxBytesReader: chi legge oltre il limite
// riceve un *http.MaxBytesError (e il server chiude la connessione dopo la risposta). Il
// ValidatorHandler lo traduce in un 413. max <= 0: nessun limite.
func LimitBody(max int64) func(http.Handler) http.Handler {
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
		return http.StatusRequestEntityTooLarge, apierrors.CodeBodyTooLarge, fmt.Sprintf("body oltre il limite di %d byte", mbe.Limit)
	}
	return http.StatusBadRequest, apierrors.CodeBodyRead, "body della richiesta non leggibile"
}
