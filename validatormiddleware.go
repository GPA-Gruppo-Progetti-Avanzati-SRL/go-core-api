package apiservices

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"reflect"
	"time"

	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/go-core-app"
	"github.com/danielgtaylor/huma/v2"
	"github.com/rs/zerolog/log"

	"github.com/go-playground/validator/v10"
)

func (r *Router) ValidatorHandler(ctx huma.Context, next func(huma.Context)) {

	vc := &ValidatorContext{c: ctx}

	if vc.Operation().RequestBody == nil {
		next(vc)
		return
	}
	registry := r.Api.OpenAPI().Components.Schemas

	content, ok := ctx.Operation().RequestBody.Content["application/json"]
	if !ok {
		next(vc)
		return
	}
	schemaRef := content.Schema.Ref
	t := registry.TypeFromRef(schemaRef)
	log.Trace().Msgf("ValidatorHandler type: %+v", t)
	input := reflect.New(t).Interface()
	b, err := io.ReadAll(vc.BodyReader())
	if err != nil {
		next(vc)
		return
	}
	if berr := json.Unmarshal(b, input); berr != nil {
		next(vc)
		return
	}

	log.Trace().Msgf("ValidatorHandler Input: %+v", input)
	// Valida i dati se non sono nulli
	if input != nil {
		if verr := r.Validator.Struct(input); verr != nil {
			var errValidate validator.ValidationErrors
			var errorMessages []string
			var errmsg string

			vc.SetStatus(400)
			vc.SetHeader("Content-Type", "application/json")
			log.Debug().Err(verr).Msg("Validation error")

			if errors.As(verr, &errValidate) {
				for _, everr := range errValidate {
					errorMessages = append(errorMessages, fmt.Sprintf("Field '%s': %s.", everr.Field(), everr.Translate(r.Tranlator.GetFallback())))
				}
				errmsg = fmt.Sprintf("Validation errors: %s", errorMessages)
			} else {
				errmsg = fmt.Sprintf("Validation error: %s", verr.Error())
			}

			er := core.TechnicalErrorWithCodeAndMessage(ErrValidation, errmsg)
			bitErrResposnse, _ := json.Marshal(er)
			vc.BodyWriter().Write(bitErrResposnse)
			return
		}
	}
	// Procede con il prossimo handler
	next(vc)

}

type ValidatorContext struct {
	c  huma.Context
	br *bytes.Reader
}

func (r *ValidatorContext) TLS() *tls.ConnectionState {
	return r.c.TLS()

}

func (r *ValidatorContext) Version() huma.ProtoVersion {
	return r.c.Version()
}

func (r *ValidatorContext) Operation() *huma.Operation {
	return r.c.Operation()
}

func (r *ValidatorContext) Host() string {
	return r.c.Host()
}

func (r *ValidatorContext) RemoteAddr() string {
	return r.c.RemoteAddr()
}

func (r *ValidatorContext) URL() url.URL {
	return r.c.URL()
}

func (r *ValidatorContext) Param(name string) string {
	return r.c.Param(name)
}

func (r *ValidatorContext) Query(name string) string {
	return r.c.Query(name)
}

func (r *ValidatorContext) Header(name string) string {
	return r.c.Header(name)
}

func (r *ValidatorContext) EachHeader(cb func(name string, value string)) {
	r.c.EachHeader(cb)
}

func (r *ValidatorContext) BodyReader() io.Reader {

	if r.br != nil {
		r.br.Seek(0, 0)
		return r.br
	}
	b, _ := io.ReadAll(r.c.BodyReader())
	r.br = bytes.NewReader(b)
	return r.br

}

func (r *ValidatorContext) GetMultipartForm() (*multipart.Form, error) {
	return r.c.GetMultipartForm()
}

func (r *ValidatorContext) SetReadDeadline(time time.Time) error {
	return r.c.SetReadDeadline(time)
}

func (r *ValidatorContext) SetStatus(code int) {
	r.c.SetStatus(code)
}

func (r *ValidatorContext) Status() int {
	return r.c.Status()
}

func (r *ValidatorContext) SetHeader(name, value string) {
	r.c.SetHeader(name, value)
}

func (r *ValidatorContext) AppendHeader(name, value string) {
	r.c.AppendHeader(name, value)
}

func (r *ValidatorContext) Method() string {
	return r.c.Method()
}

func (r *ValidatorContext) BodyWriter() io.Writer {
	return r.c.BodyWriter()
}
func (r *ValidatorContext) Context() context.Context {
	return r.c.Context()
}
