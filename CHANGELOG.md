# Changelog

Tutte le modifiche note di tutti i tag della libreria.

Formato: Versione (Data) — elenco dei commit inclusi tra il tag precedente e quello corrente.

## Non rilasciato
- **Breaking (visibilità fx):** le registrazioni stanno in un `core.ModuleClosed("api")` — l'API è un sottosistema chiuso: `*Config`, `*Router`, `*chi.Mux` e il server HTTP sono privati al modulo e **non più iniettabili** dal grafo dell'app. Il `*Router` continua ad arrivare alle `Register` come parametro (il solo modo in cui serve all'app), e il business dell'app, fornito a root, resta risolvibile dagli invoke del modulo. Un'app che iniettava `*coreapi.Config` o `*chi.Mux` da un proprio costruttore non parte più (`missing type`). Richiede `go-core-app` con `core.ModuleClosed`.
- **Breaking (nome del package):** il package radice si chiama `coreapi`, non più `apiservices`, per allineamento con `coremongo`/`coresql`/`corekafka`. Il path del modulo è invariato. Le app che importano con alias esplicito (`api "…/go-core-api"`) non richiedono modifiche; la convenzione nuova è l'import non aliasato → `coreapi.Module(...)`.
- `Operation`, `Response`, `MediaType`, `Schema`, `FormFile` e `MultipartFormFiles[T]` sono riesportati da `coreapi` come **alias** dei tipi huma: un'applicazione può descrivere le proprie operazioni senza importare `github.com/danielgtaylor/huma/v2` né dichiararlo come require diretto (diventa `// indirect`). Essendo alias e non defined type, `RegisterWithBusiness` continua ad accettare le `huma.Operation` scritte a mano dalle app non migrate, e il match per reflection del multipart resta valido.
- `RegisterWithBusiness` merge le `DefaultResponses` nell'operazione registrata (copia difensiva di mappa e `*huma.Response`, le chiavi dell'op vincono): i file operazione dell'app non hanno più bisogno di `var xxxResponses` + `init()` con `maps.Copy`. Retrocompatibile con le app che lo fanno ancora.

## v0.0.10 — 2025-09-29
- b417013 — Reorder imports and move `/metrics` handler initialization inside `OnStart`. Add `/health` endpoint.

## v0.0.9 — 2025-09-23
- 1bad32b — update rep

## v0.0.8 — 2025-09-23
- 299bc37 — update rep

## v0.0.7 — 2025-07-31
- 69c1bce — Add IdleTimeout configuration for HTTP server

## v0.0.6 — 2025-05-15
- ddd1fee — Set response status before setting headers in validation.

## v0.0.5 — 2025-04-18
- 614ea97 — Update go-core-app dependency to stable version
- dc3a7da — validation on core lib
- 84091c0 — update lib and validation error

## v0.0.4 — 2025-04-01
- 2dca1e9 — log trace validation
- 5d36d46 — enhance validation error handling to include detailed messages and improve logging
- 5195da4 — add localization support and improve validation error handling
- afa87f5 — validator on router

## v0.0.3 — 2025-03-19
- 201c4ab — align to go core app
- 612484e — align to go core app

## v0.0.2 — 2025-03-11
- 731ccb1 — upgrade to go core app 0.0.2
- 413cb7d — upgrade to go core app 0.0.2
- 7b37115 — paging api
- 05b9771 — security fix key name
- a860ce8 — security
- ce83bfb — some fixes
- 61d6bd4 — openapi + metrics path
- d9dfad8 — openapi + metrics path
- 7c3d1d9 — proxies + swagger
- de1d744 — not found on default responses
- 20bb0d3 — not found on default responses
- f05848d — revert unwrap context
- a17dc94 — resolve conflicts
- 829cdcb — upgrade huma

## v0.0.1 — 2025-02-03
- 3ba23ef — Upgrade go-core-app to 0.0.1
- 12936f4 — remove hooks for schema creation on openapi thanks to fededomm
- b8abe34 — upgrade lib
- 3576354 — add description in configuration
- e383e0f — server list in api file
- 30c49c3 — Create LICENSE
- da5e4ab — rename package
- d8b4dd5 — Refactor validation error handling and introduce constants
- ff356ae — Refactor validation error handling in middleware.
- 2a91f46 — Add validator middleware for request validation
- b49f790 — init

—
Generato il 2025-11-13.
