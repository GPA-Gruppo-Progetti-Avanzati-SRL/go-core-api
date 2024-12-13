package api

import (
	"errors"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/go-core-app"
	"github.com/danielgtaylor/huma/v2"
)

const ApplicationJson = "application/json"
const ErrValidation = "ERR_VALIDATION"

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

var ErrorContent = map[string]*huma.MediaType{ApplicationJson: {
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

func ConfigureError() {
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
				Code:    ErrValidation,
				Message: message + " " + errors.Join(errs...).Error(),
			}
		}
		return orig(status, message, errs...)
	}
}
