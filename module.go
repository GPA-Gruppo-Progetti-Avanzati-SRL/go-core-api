package apiservices

import (
	core "github.com/GPA-Gruppo-Progetti-Avanzati-SRL/go-core-app"
)

// Module wires the HTTP API (chi + Huma) into the fx application. La *Config è
// passata come parametro e fornita a fx dal Module stesso (core.Supply interno):
// l'app non deve più fare core.Supply. I costruttori concreti (newService/newRouter)
// non sono esportati: l'unico entry-point è Module().
//
// Le registrazioni sono raggruppate in un fx.Module("api") per il namespacing del
// grafo/log fx (nessun fx.Private: i provide restano visibili all'app, così *Router
// e *chi.Mux sono consumabili dai siti di registrazione delle operation Huma
// dell'app). NON c'è un Invoke interno che forza la costruzione: è l'app, registrando
// le operation su *Router (o consumandolo), a farlo costruire (lazy). L'Authorizer
// resta opzionale (Matcher, fornito dall'app se l'autorizzazione è abilitata).
// Se modes è vuoto registra sempre; altrimenti solo quando core.Mode è tra i modes indicati.
func Module(cfg *Config, modes ...string) {
	core.Module("api", func() {
		core.Supply(cfg, modes...)
		core.Provide(newService, modes...)
		core.Provide(newRouter, modes...)
	})
}
