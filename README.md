# GO-CORE-API

## Installation

    go get github.com/GPA-Gruppo-Progetti-Avanzati-SRL/go-core-api

---

Framework HTTP delle applicazioni GPA (package `coreapi`): router [chi v5](https://go-chi.io/),
OpenAPI via [Huma v2](https://huma.rocks/), middleware autorizzativo RBAC, metriche e tracing,
paginazione, Swagger UI, reverse proxy.

Dipende da [`go-core-app`](../go-core-app). **Richiede Go 1.27+.**

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
func ManageBusinessError(e *core.ApplicationError) error
```

Traduce l'`ApplicationError` di go-core-app nella error response Huma, con lo status del `StatusCode`
e il body `DefaultError` (`ambit`, `code`, `message`). La causa non esportata **non** finisce nel
body: `encoding/json` ignora i campi non esportati.

---

## Paginazione

```go
type listInput struct {
    coreapi.PagingRequest              // ?pagesize= &pagenumber= &sort=
    Status string `query:"status"`
}

type listOutput = coreapi.PagedResponse[models.Person]

func list(ctx context.Context, in *listInput, b bizperson.IBusiness) (*listOutput, error) {
    sort, appErr := in.GetSort()          // "name:asc,createdAt:desc" -> page.SortRequest
    if appErr != nil {
        return nil, coreapi.ManageBusinessError(appErr)
    }
    paging := page.InitPaging(nil, in.PageSize, in.PageNumber, 0)

    items, appErr := b.List(ctx, in.Status, sort, paging)
    if appErr != nil {
        return nil, coreapi.ManageBusinessError(appErr)
    }
    return coreapi.GeneratePageResponse(items, paging), nil
}
```

`PagedResponse[T]` espone i metadati come **header** di risposta (`pageSize`, `totalCount`,
`totalPages`, `currentPage`, `hasNext`, `hasPrevious`) e la lista nel body.

---

## Autorizzazione

RBAC-based. Il middleware legge i ruoli dall'header configurato e li confronta con
l'`authorization.Authorizer` di go-core-app, che va fornito al grafo (tipicamente da
`coremongo.WithAuthorization()`, che alimenta la LUT dalla collection ACL). L'Authorizer è iniettato
nel context della richiesta, così middleware e handler a valle lo recuperano senza dipendere dal Router.

```yaml
config:
  services:
    api:
      authorization:
        enabled: true
        roles-header:   X-Roles
        context-header: X-Context
        user-header:    X-User
        delimiter: ","
        guest-paths:
          - /health
          - /api/v1/public/*
```

Con `enabled: true` e nessun Authorizer nel grafo l'app **non parte** (`log.Fatal` al wiring).
Il modulo registra anche l'operazione `Token` di scambio.

### Capabilities

`coreapi.RegisterActionCapability(id, description)` dichiara una capability non legata a una rotta.
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
Lo shutdown è agganciato al lifecycle fx (`srv.Shutdown` in `OnStop`).

### Reverse proxy

Ogni voce di `proxy:` monta un `httputil.ReverseProxy` sul `mount-path`, con gli header aggiuntivi
indicati. Serve a esporre servizi legacy dietro lo stesso host dell'API.

---

## Comandi

```bash
go build ./...
go test ./...
go test -race -count=2 ./...
go vet ./...
```
