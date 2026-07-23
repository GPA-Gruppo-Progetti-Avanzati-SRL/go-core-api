package apiservices

import (
	core "github.com/GPA-Gruppo-Progetti-Avanzati-SRL/go-core-app"
)

// Module wires the HTTP API (chi + Huma) into the fx application. La *Config è
// passata come parametro e fornita a fx dal Module stesso (core.Supply interno):
// l'app non deve più fare core.Supply. I costruttori concreti (newService/newRouter)
// non sono esportati: l'unico entry-point è Module()/ModuleIf().
//
// L'Invoke su *Router forza la costruzione di Router → Mux → avvio del server HTTP
// anche prima che l'app registri le operation Huma. L'Authorizer resta opzionale
// (Matcher, fornito dall'app se l'autorizzazione è abilitata).
func Module(cfg *Config) {
	core.Supply(cfg)
	core.Provides(newService, newRouter)

}

// ModuleIf è come Module ma attivo solo quando core.Mode è tra i modes indicati.
func ModuleIf(cfg *Config, modes ...string) {
	core.SupplyIf(cfg, modes...)
	core.ProvideIf(newService, modes...)
	core.ProvideIf(newRouter, modes...)

}
