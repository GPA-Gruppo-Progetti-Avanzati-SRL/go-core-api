# Codici di errore — go-core-api

`go-core-api` è soprattutto **il punto in cui l'`ApplicationError` diventa una risposta HTTP**:
la maggior parte dei codici che il client vede arriva dai layer sottostanti (business,
`go-core-mongo`, `go-core-sql`, `go-core-app`), che dichiarano il proprio `Ambit`.

> **`Ambit` dice da quale libreria viene l'errore.** Quelli emessi qui portano
> `Ambit = "go-core-api"` (costante `coreapi.Ambit`); nel sottopackage `authorization` la
> costante è ripetuta (`ambit`) perché quel package non importa il root.

## Mapping ApplicationError → risposta HTTP

`ManageBusinessError(e *core.ApplicationError) error` (`error.go:19`) traduce lo status:

| `StatusCode` | Risposta Huma |
|---|---|
| 400 | `huma.Error400BadRequest` |
| 404 | `huma.Error404NotFound` |
| 422 | `huma.Error422UnprocessableEntity` |
| 500 | `huma.Error500InternalServerError` |
| altro | `huma.Error500InternalServerError` con messaggio `Errore Sconosciuto` |

Il body è sempre un `DefaultError`:

```json
{ "ambit": "go-core-mongo", "code": "MONGO-FILTER", "message": "..." }
```

`configureError()` sostituisce `huma.NewError`: se fra gli errori c'è un `*core.ApplicationError`
(cercato con `errors.As`, quindi anche avvolto) il body riporta **i suoi** `Ambit`/`Code`/`Message`
e il **suo** status. Il campo `cause` è non esportato: nel JSON non compare mai.

## Codici emessi dal modulo

| Codice | HTTP | Costante | Origine | Significato |
|---|---|---|---|---|
| `ERR-SORT` | 422 | `coreapi.CodeSort` | `page.go:29` (`PagingRequest.GetSort`) | query param `sort` non parsabile; l'errore di `page.ParseSort` è la causa |
| `ERR_VALIDATION` | **400** | `core.ErrValidation` | `error.go:73` | validazione della richiesta fatta da huma (422 in ingresso) riscritta con **status 400**; il messaggio unisce quello di huma e gli errori di campo |
| `API-FORBIDDEN` | 403 | `authorization.CodeForbiddenRole` | `authorization/middleware.go:135` | nessuno dei ruoli presentati abilita la rotta |
| `API-CTX-FORBIDDEN` | 403 | `authorization.CodeForbiddenCtx` | `authorization/middleware.go:139` | il context header non è autorizzato |
| `API-TOKEN-CRYPT` | 500 | `authorization.CodeTokenEncryption` | `authorization/token.go:112` | cifratura del token fallita; l'errore di `core.Encrypt` è la causa |

### Cambiamenti rispetto al censimento precedente

- **I due 403 avevano un body tutto loro** (`{"error":"forbidden","message":"..."}`): erano le
  uniche risposte d'errore dell'API senza `code`, e nessun client poteva trattarle come le
  altre. Ora escono nella stessa forma `ambit`/`code`/`message` del `DefaultError`.
- **L'errore di cifratura del token** era un `huma.Error500InternalServerError` nudo: nessun
  codice, ambit vuoto. Ora è un `ApplicationError` con `API-TOKEN-CRYPT`.
- L'ambit della validazione huma era la stringa `"VALIDATION"`; ora è `go-core-api`, e a dire
  che si tratta di validazione resta il codice.

## Risposte d'errore dichiarate nell'OpenAPI

`DefaultResponses` (`router.go`) è mergiata in ogni operazione da `RegisterWithBusiness`:
**400, 404, 408, 422, 500**, tutte con schema `DefaultError`. Le operazioni possono
**ridefinire** una chiave (es. una Description custom sul 404), mai rimuoverla.
