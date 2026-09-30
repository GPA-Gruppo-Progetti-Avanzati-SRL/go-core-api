package coreapi

import (
	"time"
)

type Config struct {
	Host string        `yaml:"host" mapstructure:"host" json:"host"`
	Port int           `yaml:"port" mapstructure:"port" json:"port"`
	Idle time.Duration `yaml:"idle" mapstructure:"idle" json:"idle"`

	// MaxHeaderValueCount limita il numero di header value accettati per
	// richiesta (net/http Server.MaxHeaderValueCount, Go 1.27+): protezione
	// contro le richieste con migliaia di header. A 0 vale il default di
	// net/http (DefaultMaxHeaderValueCount = 500).
	MaxHeaderValueCount int `yaml:"max-header-value-count" mapstructure:"max-header-value-count" json:"max-header-value-count"`

	// Timeout del server HTTP pubblico. Un valore assente (0) vale il default della libreria, un
	// valore negativo disattiva il timeout — scelta esplicita, perché senza timeout un client lento
	// tiene aperta una connessione (e una goroutine) per sempre: è la Slowloris.
	//
	//	read-header-timeout  DefaultReadHeaderTimeout (10s): tempo per ricevere gli header
	//	read-timeout         DefaultReadTimeout (1m): tempo per ricevere l'intera richiesta
	//	write-timeout        DefaultWriteTimeout (2m): tempo per scrivere la risposta; da alzare per
	//	                     le risposte lunghe (download, streaming)
	ReadHeaderTimeout time.Duration `yaml:"read-header-timeout" mapstructure:"read-header-timeout" json:"read-header-timeout"`
	ReadTimeout       time.Duration `yaml:"read-timeout" mapstructure:"read-timeout" json:"read-timeout"`
	WriteTimeout      time.Duration `yaml:"write-timeout" mapstructure:"write-timeout" json:"write-timeout"`

	// MaxBodyBytes limita il body di ogni richiesta, rotte, proxy e middleware compresi (un body più
	// grande è un 413). 0 vale DefaultMaxBodyBytes (10 MiB), un valore negativo disattiva il limite.
	// Le operazioni huma hanno anche il proprio limite (Operation.MaxBodyBytes, 1 MiB di default),
	// che si applica dentro questo.
	MaxBodyBytes int64 `yaml:"max-body-bytes" mapstructure:"max-body-bytes" json:"max-body-bytes"`

	// DevelopMode abilita gli endpoint diagnostici/discovery (/openapi, /capabilities).
	// Default false: in produzione questi path non sono esposti.
	DevelopMode bool           `yaml:"develop-mode"  mapstructure:"develop-mode"  json:"develop-mode"`
	Proxy       []*ProxyConfig `yaml:"proxy"   mapstructure:"proxy"   json:"proxy"`
	OpenApi     *OpenApiConfig `yaml:"openapi" mapstructure:"openapi" json:"openapi"`
}

// Default del server HTTP pubblico (vedi Config).
const (
	DefaultIdleTimeout       = 30 * time.Second
	DefaultReadHeaderTimeout = 10 * time.Second
	DefaultReadTimeout       = time.Minute
	DefaultWriteTimeout      = 2 * time.Minute
	DefaultMaxBodyBytes      = 10 << 20
)

// orDefault: 0 vale il default, un negativo disattiva (0 per net/http), il resto è il valore.
func orDefault[T time.Duration | int64](v, def T) T {
	switch {
	case v == 0:
		return def
	case v < 0:
		return 0
	default:
		return v
	}
}

type OpenApiConfig struct {
	ApiName     string    `yaml:"api-name" mapstructure:"api-name" json:"api-name"`
	ApiVersion  string    `yaml:"api-version" mapstructure:"api-version" json:"api-version"`
	Servers     []*Server `yaml:"api-servers" mapstructure:"api-servers" json:"api-servers"`
	Description string    `yaml:"api-description" mapstructure:"api-description" json:"api-description"`
}

type Server struct {
	Url         string `yaml:"url" mapstructure:"url" json:"url"`
	Description string `yaml:"description" mapstructure:"description" json:"description"`
}

type ProxyConfig struct {
	MountPath string `yaml:"mount-path" mapstructure:"mount-path" json:"mount-path"`
	// Url è la destinazione, `scheme://host[:porta]`; senza scheme vale http (`legacy:8080`).
	Url     string    `yaml:"url" mapstructure:"url" json:"url"`
	Headers []*Header `yaml:"headers" mapstructure:"headers" json:"headers"`
	// ResponseHeaderTimeout è l'attesa massima degli header di risposta del backend (502 oltre).
	// 0 vale DefaultProxyResponseHeaderTimeout (30s), un negativo disattiva il limite.
	ResponseHeaderTimeout time.Duration `yaml:"response-header-timeout" mapstructure:"response-header-timeout" json:"response-header-timeout"`
}

type Header struct {
	Key   string `yaml:"key" mapstructure:"key" json:"key"`
	Value string `yaml:"value" mapstructure:"value" json:"value"`
}

// AuthorizationConfig definisce il comportamento del middleware autorizzativo.
