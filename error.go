package coreapi

import (
	"errors"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/go-core-app"
	"github.com/danielgtaylor/huma/v2"
)

const ApplicationJson = "application/json"

// Ambit è la libreria di origine dell'errore: i costruttori di core riempiono Ambit con
// l'AppName, cioè con l'app che l'errore lo riceve, quindi un errore nato qui deve dirlo.
const Ambit = "go-core-api"

// Codici emessi dal modulo. Tutti finiscono nel campo `code` del DefaultError.
const (
	CodeSort = "ERR-SORT" // query param `sort` non parsabile
)

func ManageBusinessError(e *core.ApplicationError) error {

	switch e.StatusCode {
	case 400:
		return huma.Error400BadRequest(e.Message, e)
	case 404:
		return huma.Error404NotFound(e.Message, e)
	case 422:
		return huma.Error422UnprocessableEntity(e.Message, e)
	case 500:
		return huma.Error500InternalServerError(e.Message, e)
	default:
		return huma.Error500InternalServerError("Errore Sconosciuto", e)
	}
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

func configureError() {
	orig := huma.NewError
	huma.NewError = func(status int, message string, errs ...error) huma.StatusError {
		if len(errs) > 0 {
			err := errs[0]
			var ev *core.ApplicationError
			switch {
			case errors.As(err, &ev):
				return &DefaultError{
					Status:  ev.StatusCode,
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
