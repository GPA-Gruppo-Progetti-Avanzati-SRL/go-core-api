package coreapi

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/url"
	"reflect"
	"time"

	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/go-core-app"
	"github.com/danielgtaylor/huma/v2"
	"github.com/rs/zerolog/log"
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
		if verr := core.ValidateStruct(input); verr != nil {
			vc.SetHeader("Content-Type", "application/json")
			vc.SetStatus(400)
			bitErrResposnse, merr := json.Marshal(verr)
			if merr != nil {
				log.Error().Err(merr).Msg("serializzazione della risposta di validazione fallita")
				return
			}
			// Status e header sono già stati scritti: un errore qui non è più riportabile al
			// client, ma dice che la 400 non è arrivata — e chi legge i log del client vedrebbe
			// altrimenti una risposta vuota senza spiegazione.
			if _, werr := vc.BodyWriter().Write(bitErrResposnse); werr != nil {
				log.Warn().Err(werr).Msg("invio della risposta di validazione non completato")
			}
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
		// Riavvolge il body già letto perché il prossimo lettore (l'handler, dopo il
		// middleware) lo trovi da capo. Su un bytes.Reader l'errore non è raggiungibile, ma se
		// lo fosse il lettore successivo leggerebbe un body troncato senza accorgersene.
		if _, err := r.br.Seek(0, io.SeekStart); err != nil {
			log.Error().Err(err).Msg("riavvolgimento del body fallito: il prossimo lettore vedrà un body parziale")
		}
		return r.br
	}
	b, err := io.ReadAll(r.c.BodyReader())
	if err != nil {
		// Il body letto resta quello parziale: la validazione lo vedrà così, ed è l'unico
		// punto in cui si può sapere che era troncato e non semplicemente malformato.
		log.Warn().Err(err).Msg("lettura del body incompleta")
	}
	// La deadline di lettura viene tolta perché il body è già in memoria: da qui in poi
	// nessun handler deve più aspettare la rete.
	if err := r.c.SetReadDeadline(time.Time{}); err != nil {
		log.Warn().Err(err).Msg("rimozione della read deadline fallita")
	}
	r.br = bytes.NewReader(b)
	return r.br

}

func (r *ValidatorContext) GetMultipartForm() (*multipart.Form, error) {
	return r.c.GetMultipartForm()
}

func (r *ValidatorContext) SetReadDeadline(time time.Time) error {
	//Already read body so it becomes "moot" and dangerous to set a deadline
	return nil
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
