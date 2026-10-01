package coreapi

import (
	"errors"
	"net/http"
	"sync"

	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/go-core-app"
	"github.com/danielgtaylor/huma/v2"
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

// ManageBusinessError converte un *core.Error nella risposta d'errore huma, con lo status del
// core.Error. Prima conosceva solo 400/404/422/500 e ogni altro status — un 409 di conflitto, un 403
// applicativo — diventava un 500 "Errore Sconosciuto", cioè un guasto del server per una risposta
// corretta. Uno status che non è un errore HTTP (fuori da 400..599) resta un 500: un core.Error che
// arrivi qui con 200 è un difetto di chi l'ha costruito.
func ManageBusinessError(e *core.Error) error {
	status := e.StatusCode
	if status < 400 || status > 599 {
		status = http.StatusInternalServerError
	}
	return huma.NewError(status, e.Message, e)
}

var ErrorContent = map[string]*MediaType{ApplicationJson: {
	Schema: SerializeSchema(DefaultError{}),
}}

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

// configureErrorOnce: huma.NewError è una variabile di package, e configureError la avvolge. Senza
// Once ogni newRouter avvolgeva la versione già avvolta — con più router nello stesso processo, o in
// ogni test, la catena cresceva a ogni costruzione.
var configureErrorOnce sync.Once

// configureError sostituisce huma.NewError, una sola volta per processo: un *core.Error diventa un
// DefaultError col suo status, e gli errori di validazione della richiesta (422 di huma) diventano un
// 400 ERR-VALIDATION. È voluto: 400 è un input malformato, 422 resta lo status degli errori di
// business (core.BusinessError) — i due casi hanno un client diverso da cui farsi aggiustare.
func configureError() {
	configureErrorOnce.Do(installErrorHandler)
}

func installErrorHandler() {
	orig := huma.NewError
	huma.NewError = func(status int, message string, errs ...error) huma.StatusError {
		if len(errs) > 0 {
			err := errs[0]
			var ev *core.Error
			switch {
			case errors.As(err, &ev):
				// Lo status del core.Error vince, ma solo se è uno status d'errore: un core.Error
				// costruito con 0 o 200 farebbe uscire una risposta d'errore con un 200.
				st := ev.StatusCode
				if st < 400 || st > 599 {
					st = status
				}
				return &DefaultError{
					Status:  st,
					Ambit:   ev.Ambit,
					Code:    ev.Code,
					Message: ev.Message,
				}
			default:
				break
			}
		}
		if status == 422 {
			return &DefaultError{
				Status:  400,
				Code:    core.ErrValidation,
				Message: message + " " + errors.Join(errs...).Error(),
				Ambit:   Ambit,
			}
		}
		return orig(status, message, errs...)
	}
}
