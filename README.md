# GO-CORE-API

## Installation

    go get github.com/GPA-Gruppo-Progetti-Avanzati-SRL/go-core-api

---

Framework HTTP delle applicazioni GPA (package `coreapi`): router [chi v5](https://go-chi.io/),
OpenAPI via [Huma v2](https://huma.rocks/), metriche e tracing, paginazione, Swagger UI, reverse
proxy. L'autorizzazione sta in [`go-core-auth`](../go-core-auth).

Dipende da [`go-core-app`](../go-core-app). **Richiede Go 1.27+.**

---

## Struttura del modulo

La radice `coreapi` tiene ciò che un'applicazione usa per wirare l'API e scrivere un'operazione;
quello che l'app nomina per dominio sta in un package proprio, con **gli stessi nomi di simbolo**; i
meccanismi (middleware, proxy, discovery) sono in `internal/` e non fanno parte della superficie.

| Package | Contenuto |
|---|---|
| `coreapi` | `Module`, `Option`, `WithRoutes`, `WithModes`; `Router`, `RegisterWithBusiness`, `DefaultResponses`, `ApiRegistry`, `SerializeSchema`; `Config` coi sottotipi (`OpenApiConfig`, `Server`, `ProxyConfig`, `Header`) e i `Default*`; errori (`ManageBusinessError`, `DefaultError`, `ErrorContent`, `Ambit`, `Code*`); gli alias huma (`Operation`, `Response`, …) |
| `coreapi/paging` | `PagingRequest`, `PagedResponse[T]`, `GeneratePageResponse` |
| `coreapi/capability` | `CapabilityEntry`, `Capabilities(api)`, `CapabilityID`, `RegisterActionCapability` |
| `internal/errors` | forma del body d'errore e codici (riesportati dalla radice come alias) |
| `internal/middleware` | metriche, tracing, validator dei tag `validate:`, recover e limite al body |
| `internal/proxy` | reverse proxy delle voci `proxy:` |
| `internal/opem` | endpoint di discovery di develop-mode (`/capabilities*`, `/acl.mongo.js`, `/acl.sql`) |
| `internal/swagger` | pagina `/openapi` |

Nessun package importa la radice: è lei a comporli. `DefaultError`, `ProxyConfig` e `Header` sono
**alias** dei tipi interni, quindi lo schema OpenAPI si chiama ancora `DefaultError` e lo YAML di
`proxy:` è invariato. Il nome `paging` e non `page` è voluto: i file che lo usano importano anche
`go-core-app/page` (per `page.Paging`).

### Migrazione dal package piatto

| Prima | Dopo |
|---|---|
| `coreapi.PagingRequest`, `PagedResponse`, `GeneratePageResponse` | `paging.*` (`…/go-core-api/paging`) |
| `coreapi.CapabilityEntry`, `Capabilities`, `CapabilityID`, `RegisterActionCapability` | `capability.*` (`…/go-core-api/capability`) |
| `coreapi.WithBusiness` | rimosso: `RegisterWithBusiness` è l'unico sito di registrazione |
| `coreapi.MetricsReporter`, `MetricsContext`, `ValidatorContext`, `Router.ValidatorHandler`, `NewReverseProxy`, package `swagger` | rimossi dalla superficie (in `internal/`): il Module li monta da sé |

Il resto della radice è invariato. Un'app che importa go-core-api con alias (`api "…/go-core-api"`)
cambia solo i selettori della tabella.

---

## Wiring

`coreapi.Module(cfg *Config, opts ...Option)` è **l'unico entry-point**: fornisce la `*Config` a fx
(`core.Supply` interno), il router chi + Huma (`*Router`) e il server HTTP che lo avvolge. I
costruttori concreti non sono esportati.

```go
func main() {
    svc := core.Boot[app.Config, services.Config](core.App{ /* ... */ })

    coreapi.Module(&svc.Api,
        coreapi.WithRoutes(routeperson.Register),   // func(*coreapi.Router, bizperson.IBusiness)
        coreapi.WithRoutes(routeorder.Register),
        coreapi.WithModes(engine.Api),
    )

    core.Run(core.WithTracing())
}
```

| Opzione | Effetto |
|---|---|
| `WithRoutes[B](register func(*Router, B))` | un sito di registrazione delle rotte, con il business che gli serve |
| `WithModes(modes...)` | registra solo quando `core.Mode` è tra i modes indicati; vuoto = sempre |

**L'ordine delle Option è indifferente.**

Le registrazioni sono raggruppate in un `core.ModuleClosed("api")`: l'API è un **sottosistema chiuso**
— consuma i seam dell'app e non le espone nulla in cambio. `*Config`, `*Router`, `*chi.Mux` e il
server HTTP sono **privati al modulo** e non iniettabili dal grafo dell'app: il `*Router` arriva alle
`Register` come parametro, che è il solo modo in cui serve.

### Le rotte arrivano dalle Option, insieme al loro business

`B` è inferito a compile-time dalla firma della `Register` del package rotta, e l'istanza è risolta
da fx **per tipo** (l'app la fornisce come sempre con `core.ProvideAs[B]`). Un business non
provveduto fa **fallire l'avvio** con un `missing type` di fx, mai un nil silenzioso.

Servono più dipendenze in un solo sito, o due istanze dello stesso tipo di interfaccia? `B` è un
param object:

```go
type deps struct {
    core.In
    Person bizperson.IBusiness
    Audit  bizaudit.IBusiness `name:"audit"`
}

func Register(r *coreapi.Router, d deps) { ... }
```

Un sito senza business si esprime con una `struct{ core.In }` vuota.

> L'app **non** dichiara più un tipo `routes.Router` wrapper con il suo `NewRouter(Params)` +
> `core.Provide`: quel giro esisteva solo per creare il nodo fx che innescava le registrazioni, e ora
> lo fa il Module. Senza alcun `WithRoutes` non c'è nessun invoke: il Router si costruisce solo se
> qualcuno lo consuma.

---

## Registrare un'operazione

Una operazione per file, sotto `routes/<risorsa>/`:

```go
// routes/person/register.go
func Register(r *coreapi.Router, b bizperson.IBusiness) {
    coreapi.RegisterWithBusiness(r, b, getPersonOp, getPerson)
    coreapi.RegisterWithBusiness(r, b, listPeopleOp, listPeople)
}
```

```go
// routes/person/get.go
var getPersonOp = coreapi.Operation{
    OperationID: "get-person",
    Method:      http.MethodGet,
    Path:        "/api/v1/people/{id}",
    Summary:     "Legge una persona",
    Tags:        []string{"people"},
}

type getPersonInput struct {
    Id string `path:"id"`
}

type getPersonOutput struct {
    Body *models.Person
}

func getPerson(ctx context.Context, in *getPersonInput, b bizperson.IBusiness) (*getPersonOutput, error) {
    p, appErr := b.GetById(ctx, in.Id)
    if appErr != nil {
        return nil, coreapi.ManageBusinessError(appErr)
    }
    return &getPersonOutput{Body: p}, nil
}
```

`RegisterWithBusiness` è **l'unico sito di registrazione supportato**: prende il `*Router` (non la
sola `huma.API`) così il sito dipende da fx dal Router — questo ne forza il wiring lazy e mode-gated,
garantisce la registrazione a wiring completato, e mantiene possibile più router nello stesso processo.

### Le response d'errore standard le mette la libreria

Le `coreapi.DefaultResponses` sono mergiate dentro l'operazione al momento della registrazione:

| Codice | Descrizione | Schema |
|---|---|---|
| `400` | BadRequest/Validation Error | `DefaultError` |
| `404` | Not Found | `DefaultError` |
| `408` | Request Timeout | — |
| `422` | KO Applicativo | `DefaultError` |
| `500` | Internal Server Error | `DefaultError` |

I file operazione dell'app **non** dichiarano più una `var xxxResponses` con un `init()` che fa
`maps.Copy(..., coreapi.DefaultResponses)`, e non dichiarano nemmeno il codice di successo (lo
sintetizza huma dal tipo di `Output.Body`). Il campo `Responses` sull'operazione serve solo a
**ridefinire** un codice (es. una Description custom sul 404): le chiavi dell'op vincono sui default,
ma non possono rimuoverli.

Il merge sta nella libreria — non nell'app — perché `huma.Register` **scrive dentro** `op.Responses`
(schema di `Output.Body`, header, Description dello status di successo): la mappa va copiata a ogni
registrazione, altrimenti la seconda operazione erediterebbe lo schema della prima, e con
registrazioni concorrenti sarebbe un `concurrent map writes`.

### L'app non nomina huma

I tipi che servono a descrivere un'operazione sono riesportati da `coreapi` come **alias** dei
corrispondenti huma (`huma.go`):

| Alias | huma |
|---|---|
| `coreapi.Operation` | `huma.Operation` |
| `coreapi.Response` | `huma.Response` |
| `coreapi.MediaType` | `huma.MediaType` |
| `coreapi.Schema` | `huma.Schema` |
| `coreapi.FormFile` | `huma.FormFile` |
| `coreapi.MultipartFormFiles[T]` | `huma.MultipartFormFiles[T]` |

Un'app scrive quindi `coreapi.Operation{...}` e non importa `huma/v2`, che nel suo `go.mod` resta un
require `// indirect` — stesso ruolo che `core.In`/`core.Out` hanno per fx in go-core-app.

Sono **alias** (`=`), non defined type, per tre ragioni: (a) nessuna conversione da tenere allineata
al variare di huma; (b) `RegisterWithBusiness` continua ad accettare le `huma.Operation` delle app
non ancora migrate; (c) per il multipart è **l'unica forma possibile** — huma riconosce la richiesta
per reflection sul tipo concreto di `RawBody`, quindi una struct propria farebbe ricadere sul path
raw body (`[]byte` → `reflect.Value.SetBytes` → panic a runtime).

```go
type uploadInput struct {
    RawBody coreapi.MultipartFormFiles[struct {
        File coreapi.FormFile `form:"file" contentType:"application/octet-stream"`
        Meta string           `form:"metadata" required:"true"`
    }]
}
```

`Router.Api` resta di tipo `huma.API`: le app quel tipo non lo nominano mai, lo passano soltanto.

---

## Errori

Codici emessi dalla libreria: **[ERRORI.md](ERRORI.md)** (`coreapi.Ambit` = `"go-core-api"`). Nota che
il **403 del middleware di autorizzazione** ha ora la stessa forma degli altri errori — `DefaultError`
con `ambit`/`code`/`message`, codici `API-FORBIDDEN` e `API-CTX-FORBIDDEN` — mentre prima era un
`{"error":"forbidden",...}` senza codice, l'unica risposta d'errore che un client non potesse trattare
come le altre.

```go
func ManageBusinessError(e *core.Error) error
```

Traduce il `core.Error` di go-core-app nella error response Huma, con lo status del `StatusCode`
— **qualunque** status d'errore (prima solo 400/404/422/500 passavano, e un 409 o un 403 diventavano un
500 "Errore Sconosciuto"); uno status fuori da 400..599 vale 500 — e il body `DefaultError` (`ambit`,
`code`, `message`). La causa non esportata **non** finisce nel body: `encoding/json` ignora i campi non
esportati.

Gli errori di **validazione della richiesta** di huma (il suo 422) escono come **400 `ERR-VALIDATION`**,
ed è voluto: 400 è un input malformato, 422 resta lo status degli errori di business
(`core.BusinessError`), e i due casi chiedono al client cose diverse. La conversione si installa una
volta sola per processo, anche con più router.

---

## Paginazione

```go
type listInput struct {
    paging.PagingRequest               // ?pagesize= &pagenumber= &sort=
    Status string `query:"status"`
}

type listOutput = paging.PagedResponse[models.Person]

func list(ctx context.Context, in *listInput, b bizperson.IBusiness) (*listOutput, error) {
    sort, appErr := in.GetSort()          // "name:asc,createdAt:desc" -> page.SortRequest
    if appErr != nil {
        return nil, coreapi.ManageBusinessError(appErr)
    }
    pg := page.InitPaging(nil, in.PageSize, in.PageNumber, 0)

    items, appErr := b.List(ctx, in.Status, sort, pg)
    if appErr != nil {
        return nil, coreapi.ManageBusinessError(appErr)
    }
    return paging.GeneratePageResponse(items, pg), nil
}
```

`PagedResponse[T]` espone i metadati come **header** di risposta (`pageSize`, `totalCount`,
`totalPages`, `currentPage`, `hasNext`, `hasPrevious`) e la lista nel body.

---

## Autorizzazione

**Non è di questo modulo, e questo modulo non la conosce**: engine, sorgenti di ACL e middleware
stanno in `go-core-auth`, che dipende da qui e non viceversa — un middleware è un plugin del
framework HTTP, e il framework non conosce i suoi plugin. `go-core-api` dipende dal solo
`go-core-app`.

Il middleware si monta da sé attraverso `WithRoutes`, senza che serva un'Option dedicata: per quel
seam è un business come un altro.

```go
coreauth.Module(&svc.Auth,
    coreauth.WithSource(mongosource.Module),
    coreauth.WithMiddleware(apiauth.Module))   // fornisce il *Middleware a fx

coreapi.Module(&svc.Api,
    coreapi.WithRoutes(apiauth.Register),      // func(*Router, *apiauth.Middleware)
    coreapi.WithRoutes(routes.Register))
```

Che l'autorizzazione sia attiva lo decide la configurazione di go-core-auth
(`services.auth.middleware.enabled`), non quella dell'API: prima la stessa decisione stava in due
posti. Se non è attiva, `apiauth.Register` non monta nulla e lo dice con un log.

`apiauth.Register` registra anche l'operazione `Token` (`GET /api/token`) via
`RegisterWithBusiness`, quindi con le response d'errore standard di questo modulo. Dentro un
handler, identità e ruoli si leggono con gli accessor di `apiauth` (`UserFrom`, `RolesFrom`,
`ContextIDFrom`, `AuthorizerFrom`).

### Capabilities

`capability.RegisterActionCapability(id, description)` dichiara una capability non legata a una rotta.
In `develop-mode` il server espone gli endpoint di discovery, utili per generare il seed dell'ACL:

| Path | Contenuto |
|---|---|
| `/capabilities` | JSON delle capability derivate dalle operazioni registrate |
| `/capabilities.yaml` | stessa lista in YAML |
| `/acl.mongo.js` | script di seed per la collection ACL Mongo |
| `/acl.sql` | INSERT di seed per il backend SQL |
| `/openapi` | Swagger/Scalar UI |
| `/debug/pprof/*` | profili runtime (`goroutine`, `goroutineleak`, `heap`, `profile`, …) |

Sono registrati direttamente su chi: niente auth, e non appaiono nella spec OpenAPI.

---

## Configurazione

```yaml
config:
  services:
    api:
      host: ""
      port: 8080
      idle: 30s                      # default 30s
      read-header-timeout: 10s       # default 10s  (negativo = disattivato)
      read-timeout: 1m               # default 1m
      write-timeout: 2m              # default 2m   (da alzare per download/streaming lunghi)
      max-body-bytes: 10485760       # default 10 MiB (negativo = nessun limite)
      develop-mode: false            # true = /openapi + endpoint di discovery
      max-header-value-count: 0      # 0 = default net/http (500)
      openapi:
        api-name: my-service
        api-version: v1
        api-description: "..."
        api-servers:
          - url: https://api.example.com
            description: prod
      authorization:
        enabled: true
        roles-header: X-Roles
      proxy:
        - mount-path: /legacy
          url: http://legacy-service:8080
          response-header-timeout: 30s   # default
          headers:
            - key: X-Forwarded-By
              value: my-service
```

**`develop-mode` è false di default**: in produzione `/openapi` e gli endpoint di discovery non sono
esposti, e lo `SchemaLinkTransformer` di huma è disattivato insieme a loro.

Per `/debug/pprof/*` questo è l'**unico** gate, e non è un dettaglio: qui la porta è quella
**pubblica** dell'API, condivisa con le rotte applicative, quindi un pprof sempre acceso
regalerebbe a chiunque raggiunga il servizio `/debug/pprof/profile?seconds=N` (CPU-burn) e
`/debug/pprof/heap` (può contenere segreti). Nei processi **senza** API il gate è invece
`metrics.pprof: true` di go-core-app, sul server ops `:2112`.

`max-header-value-count` è il `Server.MaxHeaderValueCount` di net/http (Go 1.27+): protezione contro
le richieste con migliaia di header.

**Il server pubblico ha timeout e un limite al body di default.** Senza, un client lento teneva
aperta una connessione — e una goroutine — per sempre (Slowloris), e ogni body veniva letto per
intero in memoria. I tre timeout valgono il default se assenti e si disattivano solo con un valore
**negativo**, scelto apposta; `max-body-bytes` si applica a ogni richiesta, proxy compresi, e un body
più grande riceve un **413** (`API-BODY-TOO-LARGE`). Le operazioni huma hanno anche il proprio limite
(`Operation.MaxBodyBytes`, 1 MiB di default), che resta valido dentro questo.

**Un panic in un handler diventa un 500** in forma `DefaultError` (`API-PANIC`), loggato con lo stack
e la richiesta che l'ha causato: prima net/http chiudeva la connessione, e il client vedeva una
risposta vuota senza che nulla finisse nelle metriche. `http.ErrAbortHandler`, con cui un handler o il
reverse proxy interrompono apposta una risposta, viene rilanciato.

**Il validator** (`internal/middleware`, i tag `validate:` del body) salta i body il cui schema non ha un tipo con
nome — un array o un tipo inline al top-level, che prima lo facevano panicare — e lascia a huma la
validazione dello schema; un body che non si riesce a leggere per intero ferma la richiesta (413 o
`API-BODY-READ` 400) invece di arrivare all'handler troncato.

---

## Server HTTP

Il modulo avvia il server sulla porta configurata ed espone, oltre alle rotte dell'app:

| Path | Contenuto |
|---|---|
| `/metrics` | metriche Prometheus (richieste, latenze, status) |
| `/health` | health check |

> Per questo `core.WithServerMetrics` **non** va usata in mode API: `/metrics` è già servito qui,
> sulla porta dell'API.

Ogni richiesta passa per il middleware di metriche, quello di tracing OTel e il validatore.

Lo span porta metodo, route e status, più **una lista chiusa di header** (`content-type`, `accept`,
`user-agent`, `x-request-id`, `x-forwarded-for`, come `http.request.header.*`): prima ci finivano
**tutti**, quindi `Authorization`, `Cookie` e i token applicativi arrivavano in chiaro al backend di
tracing. Lo span è in errore solo sui **5xx**: un 4xx è una risposta corretta a una richiesta
sbagliata (prima ogni status ≥ 300 lo marcava in errore).
Lo shutdown è agganciato al lifecycle fx (`srv.Shutdown` in `OnStop`).

### Reverse proxy

Ogni voce di `proxy:` monta un `httputil.ReverseProxy` sul `mount-path`, con gli header aggiuntivi
indicati. Serve a esporre servizi legacy dietro lo stesso host dell'API.

- **`url`** è `scheme://host[:porta]`: `https://legacy:8443` parla TLS col backend, `legacy:8080` (senza
  scheme, la forma storica) vale http. Prima lo scheme era `http` fisso, quindi un backend https
  riceveva la richiesta in chiaro. Un `url` non valido — vuoto, scheme diverso da http/https, **con un
  path**, che verrebbe ignorato — **ferma l'avvio**.
- **`response-header-timeout`** (default 30s, negativo = nessun limite) è l'attesa massima della
  risposta del backend, oltre la quale il client riceve 502. Il transport di default non ne aveva una:
  un backend che accetta la connessione e tace teneva occupata la richiesta fino al `write-timeout`.
- Il proxy ha il proprio **span OTel** e propaga il contesto di tracing al backend.
- **Non attraversa i middleware huma**: metriche huma, validazione e **autorizzazione** non si
  applicano, perché il proxy è montato sul mux chi e non è un'operazione. Valgono il recover e il limite
  al body. Un backend esposto così è raggiungibile da chiunque raggiunga l'API: se serve un controllo
  d'accesso, è del backend.

---

## Comandi

```bash
go build ./...
go test ./...
go test -race -count=2 ./...
go vet ./...
```
