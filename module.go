package coreapi

import (
	core "github.com/GPA-Gruppo-Progetti-Avanzati-SRL/go-core-app"
)

// Option configura Module. L'ordine in cui si passano le Option è indifferente:
// le registrazioni delle rotte sono closure differite che ricevono i modes solo
// quando Module le applica, quindi WithModes può stare prima o dopo i WithRoutes.
type Option func(*options)

type options struct {
	modes  []string
	routes []func(modes []string)
}

// WithModes limita le registrazioni dell'API ai core.Mode indicati. Vuoto = sempre attive.
func WithModes(modes ...string) Option {
	return func(o *options) { o.modes = modes }
}

// WithRoutes associa un sito di registrazione delle rotte al business che gli serve.
// register è la Register(r, b) del package rotta: il tipo B è inferito dalla sua firma
// a compile-time (nessuna stringa, nessun cast) e l'istanza è risolta da fx PER TIPO dal
// grafo (l'app la fornisce come sempre con core.ProvideAs[B]). Un business non provveduto
// fa fallire l'avvio con un "missing type" di fx, mai un nil silenzioso.
//
//	apiservices.Module(&cfg.ApiConfig,
//	    apiservices.WithRoutes(routeperson.Register),   // func(*Router, bizperson.IBusiness)
//	    apiservices.WithRoutes(routeorder.Register),
//	    apiservices.WithModes(engine.Api),
//	)
//
// Servono più dipendenze (o due istanze dello stesso tipo di interfaccia) in un solo sito?
// B è un param object `struct{ core.In; ... }`, dove i tag `name:"..."` hanno un posto dove
// stare. Un sito senza business si esprime con una `struct{ core.In }` vuota.
//
// La dipendenza da *Router nell'invoke non è cerimonia: (a) forza la costruzione del Router,
// che è lazy e mode-gated, e a cascata quella del server HTTP (newService); (b) garantisce che
// la registrazione avvenga a wiring completato; (c) mantiene possibile più router nello stesso
// processo. Vedi il doc-comment di RegisterWithBusiness.
func WithRoutes[B any](register func(*Router, B)) Option {
	return func(o *options) {
		o.routes = append(o.routes, func(modes []string) {
			invoke(func(r *Router, b B) { register(r, b) }, modes...)
		})
	}
}

// invoke indirizza la registrazione fx: var di package per poterla stubbare nei test.
var invoke = core.Invoke

// Module wira l'HTTP API (chi + Huma) nell'applicazione fx. È l'unico entry-point: la *Config è
// passata come parametro e fornita a fx dal Module stesso (core.Supply interno), i costruttori
// concreti (newService/newRouter) non sono esportati, e le rotte dell'app arrivano dalle Option
// WithRoutes — quindi l'app non deve più dichiarare un tipo Router wrapper con il suo costruttore
// solo per creare il nodo fx che innesca le registrazioni.
//
// Le registrazioni sono raggruppate in un core.ModuleClosed("api"): l'API è un sottosistema chiuso
// — consuma i seam dell'app (le rotte con il loro business) e non le espone nulla in cambio, quindi
// *Config, *Router, *chi.Mux e il server HTTP sono privati al modulo e non iniettabili dal grafo
// dell'app (il *Router arriva alle Register come parametro, che è il solo modo in cui serve). Il
// business dell'app è fornito a root e resta risolvibile dagli invoke del modulo, che ne è
// discendente. L'Authorizer resta opzionale (Matcher, fornito dall'app se l'autorizzazione è abilitata).
//
// Senza alcun WithRoutes non c'è nessun invoke: il Router (e con lui il server) si costruisce solo
// se qualcuno lo consuma — comportamento storico, invariato.
func Module(cfg *Config, opts ...Option) {
	var o options
	for _, opt := range opts {
		opt(&o)
	}

	core.ModuleClosed("api", func() {
		core.Supply(cfg, o.modes...)
		core.Provide(newService, o.modes...)
		core.Provide(newRouter, o.modes...)
		for _, reg := range o.routes {
			reg(o.modes)
		}
	})
}
