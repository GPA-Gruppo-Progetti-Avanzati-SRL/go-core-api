package apiservices

import (
	"encoding/json"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/go-core-app"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog/log"
	"io"
	"reflect"
)

var validate = validator.New()

func (r *Router) ValidatorHandler(ctx huma.Context, next func(huma.Context)) {

	if ctx.Operation().RequestBody == nil {
		next(ctx)
		return
	}
	registry := r.Api.OpenAPI().Components.Schemas
	req, res := humachi.Unwrap(ctx)
	content, ok := ctx.Operation().RequestBody.Content["application/json"]
	if !ok {
		next(ctx)
		return
	}
	schemaRef := content.Schema.Ref
	t := registry.TypeFromRef(schemaRef)
	log.Info().Msgf("ValidatorHandler type: %+v", t)
	input := reflect.New(t).Interface()
	reqClone := req.Clone(ctx.Context())
	b, err := io.ReadAll(reqClone.Body)
	if err != nil {
		next(ctx)
		return
	}
	if berr := json.Unmarshal(b, input); berr != nil {
		next(ctx)
		return
	}

	log.Info().Msgf("ValidatorHandler Input: %+v", input)
	// Valida i dati se non sono nulli
	if input != nil {
		if errValidate := validate.Struct(input); errValidate != nil {
			log.Warn().Err(err).Msg("Validation error")
			res.WriteHeader(400)
			res.Header().Add("Content-Type", "application/json")
			er := core.TechnicalErrorWithCodeAndMessage(ErrValidation, errValidate.Error())
			bitErrResposnse, _ := json.Marshal(er)
			res.Write(bitErrResposnse)
			return
		}
	}

	// Procede con il prossimo handler
	next(ctx)

}
