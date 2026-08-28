# Codici di errore — go-core-api

`go-core-api` non è tanto un produttore di codici quanto **il punto in cui l'`ApplicationError`
diventa una risposta HTTP**. I codici che il client vede sono per la quasi totalità quelli
generati dai layer sottostanti (business, `go-core-mongo`, `go-core-sql`, `go-core-app`).

## Mapping ApplicationError → risposta HTTP

`ManageBusinessError(e *core.ApplicationError) error` (`error.go:11`) traduce lo status:

| `StatusCode` | Risposta Huma |
|---|---|
| 400 | `huma.Error400BadRequest` |
| 404 | `huma.Error404NotFound` |
| 422 | `huma.Error422UnprocessableEntity` |
| 500 | `huma.Error500InternalServerError` |
| altro | `huma.Error500InternalServerError` con messaggio `Errore Sconosciuto` |

Il body è sempre un `DefaultError` (`error.go:31`):

```json
{ "ambit": "...", "code": "...", "message": "..." }
```

`configureError()` sostituisce `huma.NewError`: se fra gli errori c'è un `*core.ApplicationError`
(cercato con `errors.As`, quindi anche avvolto) il body riporta **i suoi** `Ambit`/`Code`/`Message`
e il **suo** status, non quello passato a huma. Il campo `cause` è non esportato: nel JSON non
compare mai.

## Codici emessi dal modulo

| Codice | HTTP | Origine | Significato |
|---|---|---|---|
| `ERR-SORT` | 422 | `page.go:29` (`PagingRequest.GetSort`) | parametro `sort` non parsabile; l'errore di `page.ParseSort` è allegato come causa |
| `ERR_VALIDATION` | **400** | `error.go:65` | validazione della richiesta fatta da huma (status 422 in ingresso) riscritta con `Ambit: VALIDATION`, `Code: core.ErrValidation` e **status 400**: il messaggio unisce quello di huma e gli errori di campo |

## Risposte d'errore dichiarate nell'OpenAPI

`DefaultResponses` (`router.go:138`) è mergiata in ogni operazione da `RegisterWithBusiness`:
**400, 404, 408, 422, 500**, tutte con schema `DefaultError`. Le operazioni non le dichiarano
più a mano; possono solo **ridefinire** una chiave (es. una Description custom sul 404), mai
rimuoverla.

## Errori senza codice applicativo

| Condizione | Origine | Risposta |
|---|---|---|
| ruolo/capability mancante | `authorization/middleware.go:127,133` | **403 Forbidden** scritto direttamente sul context, senza body `DefaultError` |
| errore di cifratura del token | `authorization/token.go:112` | `huma.Error500InternalServerError("encryption error", err)` |
