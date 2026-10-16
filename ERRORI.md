# Codici di errore — go-core-api

`go-core-api` è soprattutto **il punto in cui il `core.Error` diventa una risposta HTTP**:
la maggior parte dei codici che il client vede arriva dai layer sottostanti (business,
`go-core-mongo`, `go-core-sql`, `go-core-app`, `go-core-auth`), che dichiarano il proprio `Ambit`.

> **`Ambit` dice da quale libreria viene l'errore.** Quelli emessi qui portano
> `Ambit = "go-core-api"` (costante `coreapi.Ambit`, definita in `internal/errors` e riesportata).

## Mapping core.Error → risposta HTTP

`ManageBusinessError(e *core.Error) error` (`error.go`) usa **lo status del `core.Error`**, qualunque
esso sia (409, 403, 413, …). Uno status fuori da 400..599 — un `core.Error` costruito con 200 —
diventa 500, perché è un difetto di chi l'ha costruito.

Il body è sempre un `DefaultError`:

```json
{ "ambit": "go-core-mongo", "code": "MONGO-FILTER", "message": "..." }
```

`configureError()` sostituisce `huma.NewError`: se fra gli errori c'è un `*core.Error`
(cercato con `errors.As`, quindi anche avvolto) il body riporta **i suoi** `Ambit`/`Code`/`Message`
e il **suo** status. Il campo `cause` è non esportato: nel JSON non compare mai.

## Codici emessi dal modulo

| Codice | HTTP | Costante | Origine | Significato |
|---|---|---|---|---|
| `ERR-SORT` | 422 | `coreapi.CodeSort` | `paging/paging.go` (`PagingRequest.GetSort`) | query param `sort` non parsabile, o con un campo che non è un identificatore (`page.ValidSortField`); l'errore di `page.ParseSort` è la causa |
| `API-PANIC` | 500 | `coreapi.CodePanic` | `internal/middleware/guard.go` (`Recoverer`) | un handler è andato in panic: la risposta è un DefaultError, lo stack e la richiesta sono nel log |
| `API-BODY-TOO-LARGE` | 413 | `coreapi.CodeBodyTooLarge` | `internal/middleware/validator.go` (`Validator`) | body oltre `max-body-bytes` |
| `API-BODY-READ` | 400 | `coreapi.CodeBodyRead` | `internal/middleware/validator.go` (`Validator`) | body della richiesta non leggibile per intero (connessione interrotta) |
| `ERR_VALIDATION` | **400** | `core.ErrValidation` | `error.go` (`installErrorHandler`) | validazione della richiesta fatta da huma (422 in ingresso) riscritta con **status 400**; il messaggio unisce quello di huma e gli errori di campo |

I 403 dell'autorizzazione e l'errore di cifratura del token **non sono più di questo modulo**:
il middleware vive in `go-core-auth/apiauth` e i codici sono `AUTH-FORBIDDEN`,
`AUTH-CTX-FORBIDDEN`, `AUTH-TOKEN-CRYPT` (vedi `go-core-auth/ERRORI.md`). Escono comunque nella
forma `DefaultError`, perché passano dallo stesso handler d'errore.

## Risposte d'errore dichiarate nell'OpenAPI

`DefaultResponses` (`router.go`) è mergiata in ogni operazione da `RegisterWithBusiness`:
**400, 404, 408, 422, 500**, tutte con schema `DefaultError`. Le operazioni possono
**ridefinire** una chiave (es. una Description custom sul 404), mai rimuoverla.
