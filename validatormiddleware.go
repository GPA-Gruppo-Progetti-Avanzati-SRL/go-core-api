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
	if !ok || content.Schema == nil || content.Schema.Ref == "" {
		// Nessuno schema con nome (un array o un tipo inline al top-level del body): non c'è un tipo
		// Go da cui rileggere i tag `validate:`, e TypeFromRef su un Ref vuoto panicava
		// (`ref[len(prefix):]`). La validazione dello schema la fa comunque huma.
		next(vc)
		return
	}
	t := registry.TypeFromRef(content.Schema.Ref)
	if t == nil || t.Kind() != reflect.Struct {
		// I tag `validate:` sono per struct: un altro tipo lo valida huma dallo schema.
		next(vc)
		return
	}
	log.Trace().Msgf("ValidatorHandler type: %+v", t)
	input := reflect.New(t).Interface()
	b, err := vc.readBody()
	if err != nil {
		// Un body non letto per intero — oltre `max-body-bytes`, o connessione interrotta — non va
		// passato all'handler: la validazione sarebbe stata saltata e l'handler avrebbe lavorato su
		// un body troncato. Prima si proseguiva con next(vc).
		status, code, msg := bodyReadError(err)
		writeHumaError(vc, status, code, msg)
		return
	}
	if berr := json.Unmarshal(b, input); berr != nil {
		// JSON non decodificabile nel tipo: lo rifiuta huma, che valida il body contro lo schema e
		// risponde con il suo errore di formato. Qui non c'è niente da aggiungere.
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
	c       huma.Context
	br      *bytes.Reader
	buf     []byte
	readErr error
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

// readBody legge il body una volta sola, lo tiene in memoria per i lettori successivi e ritorna
// l'errore di lettura: un body troncato (oltre `max-body-bytes`, o una connessione interrotta) deve
// fermare la richiesta, non arrivare all'handler come se fosse completo.
func (r *ValidatorContext) readBody() ([]byte, error) {
	if r.br != nil {
		return r.buf, r.readErr
	}
	b, err := io.ReadAll(r.c.BodyReader())
	// La deadline di lettura viene tolta perché il body è già in memoria: da qui in poi
	// nessun handler deve più aspettare la rete.
	if derr := r.c.SetReadDeadline(time.Time{}); derr != nil {
		log.Warn().Err(derr).Msg("rimozione della read deadline fallita")
	}
	r.buf, r.readErr, r.br = b, err, bytes.NewReader(b)
	return b, err
}

func (r *ValidatorContext) BodyReader() io.Reader {
	if _, err := r.readBody(); err != nil {
		log.Warn().Err(err).Msg("lettura del body incompleta")
	}
	// Riavvolto perché ogni lettore (l'handler, dopo il middleware) lo trovi da capo. Su un
	// bytes.Reader l'errore non è raggiungibile, ma se lo fosse il lettore successivo leggerebbe
	// un body troncato senza accorgersene.
	if _, err := r.br.Seek(0, io.SeekStart); err != nil {
		log.Error().Err(err).Msg("riavvolgimento del body fallito: il prossimo lettore vedrà un body parziale")
	}
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

// writeHumaError scrive un DefaultError sul context huma, per chi non passa da un handler.
func writeHumaError(ctx huma.Context, status int, code, msg string) {
	ctx.SetHeader("Content-Type", "application/json")
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
