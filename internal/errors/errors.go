// Package errors contiene la forma del body d'errore di go-core-api, i codici della libreria e i
// writer che lo serializzano per chi non passa da un handler huma (recover, middleware).
//
// È una foglia: lo importano la radice coreapi — che ne riesporta tipo e codici come alias, perché
// sono superficie pubblica — i middleware interni e paging, e non importa nessuno di loro. Senza,
// i middleware dovrebbero importare la radice che li monta.
package errors

import (
	"encoding/json"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/rs/zerolog/log"
)

const ApplicationJson = "application/json"

// Ambit è la libreria di origine dell'errore: i costruttori di core riempiono Ambit con
// l'AppName, cioè con l'app che l'errore lo riceve, quindi un errore nato qui deve dirlo.
const Ambit = "go-core-api"

// Codici emessi dal modulo. Tutti finiscono nel campo `code` del DefaultError.
const (
	CodeSort         = "ERR-SORT"           // query param `sort` non parsabile
	CodePanic        = "API-PANIC"          // 500: un handler è andato in panic (recoverer)
	CodeBodyTooLarge = "API-BODY-TOO-LARGE" // 413: body oltre `max-body-bytes`
	CodeBodyRead     = "API-BODY-READ"      // 400: body della richiesta non leggibile
)

type DefaultError struct {
	Status  int    `json:"-"`
	Ambit   string `json:"ambit"`
	Code    string `json:"code" yaml:"code"`
	Message string `json:"message" yaml:"message"`
}

func (e *DefaultError) Error() string {
	return e.Message
}

func (e *DefaultError) GetStatus() int {
	return e.Status
}

// Write scrive un DefaultError della libreria su un http.ResponseWriter.
func Write(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", ApplicationJson)
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(&DefaultError{Ambit: Ambit, Code: code, Message: msg}); err != nil {
		log.Warn().Err(err).Str("code", code).Msg("invio della risposta d'errore non completato")
	}
}

// WriteHuma scrive un DefaultError sul context huma, per chi non passa da un handler.
func WriteHuma(ctx huma.Context, status int, code, msg string) {
	ctx.SetHeader("Content-Type", ApplicationJson)
	ctx.SetStatus(status)
	b, err := json.Marshal(&DefaultError{Ambit: Ambit, Code: code, Message: msg})
	if err != nil {
		log.Error().Err(err).Str("code", code).Msg("serializzazione della risposta d'errore fallita")
		return
	}
	if _, err := ctx.BodyWriter().Write(b); err != nil {
		log.Warn().Err(err).Str("code", code).Msg("invio della risposta d'errore non completato")
	}
}
