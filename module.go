package apiservices

import (
	core "github.com/GPA-Gruppo-Progetti-Avanzati-SRL/go-core-app"
)

// Module wires the HTTP API (chi + Huma) into the fx application. La *Config è
// passata come parametro e fornita a fx dal Module stesso (core.Supply interno):
// l'app non deve più fare core.Supply. I costruttori concreti (newService/newRouter)
// non sono esportati: l'unico entry-point è Module().
//
// L'Invoke su *Router forza la costruzione di Router → Mux → avvio del server HTTP
// anche prima che l'app registri le operation Huma. L'Authorizer resta opzionale
// (Matcher, fornito dall'app se l'autorizzazione è abilitata).
// Se modes è vuoto registra sempre; altrimenti solo quando core.Mode è tra i modes indicati.
func Module(cfg *Config, modes ...string) {
	core.Supply(cfg, modes...)
	core.Provide(newService, modes...)
	core.Provide(newRouter, modes...)
}
