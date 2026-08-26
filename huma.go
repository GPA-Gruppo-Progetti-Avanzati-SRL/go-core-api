package coreapi

import "github.com/danielgtaylor/huma/v2"

// I tipi che l'app usa per descrivere le proprie operazioni, riesportati da
// go-core-api così che un'applicazione non debba importare huma né dichiararlo
// come require diretto — stesso ruolo di core.In/core.Out per fx in go-core-app.
//
// Sono ALIAS (`=`), non defined type: sono lo stesso identico tipo di huma, quindi
// (a) RegisterWithBusiness continua ad accettare una huma.Operation scritta a mano
// dalle app non ancora migrate, (b) non c'è nessuna conversione da mantenere
// allineata al variare di huma.
type (
	Operation = huma.Operation
	Response  = huma.Response
	MediaType = huma.MediaType
	Schema    = huma.Schema
	FormFile  = huma.FormFile
)

// MultipartFormFiles è il tipo che il campo RawBody DEVE avere perché huma
// riconosca la richiesta come multipart: il match avviene per reflection sul tipo
// concreto, quindi qui un alias non è una scorciatoia ma l'unica forma possibile —
// una struct propria di go-core-api farebbe ricadere la richiesta sul path raw body
// ([]byte → reflect.Value.SetBytes → panic a runtime).
//
// Alias generico: legale da Go 1.24, il modulo è go 1.27.
//
//	type CreateInput struct {
//	    RawBody coreapi.MultipartFormFiles[struct {
//	        File coreapi.FormFile `form:"file" contentType:"application/octet-stream"`
//	        Meta string           `form:"metadata" required:"true"`
//	    }]
//	}
type MultipartFormFiles[T any] = huma.MultipartFormFiles[T]
