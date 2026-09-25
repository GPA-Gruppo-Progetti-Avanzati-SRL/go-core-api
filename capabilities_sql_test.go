package coreapi

import (
	"strings"
	"testing"
)

// Qui si verifica ciò che questo modulo può verificare da solo: che /acl.sql resti al vocabolario
// del frontdoor OPEM e che il seed non assegni nulla ai ruoli.
//
// Il confronto fra le colonne generate da /acl.coreauth.sql e quelle lette da
// go-core-auth/sqlsource NON sta qui: richiederebbe di importare go-core-auth, e questo modulo non
// deve dipenderne. Sta in go-core-auth (seed_test.go), che è l'unico dei due a vedere entrambi.

// Il seed descrive ciò che l'applicazione espone; assegnare le capability a un ruolo è una
// decisione di chi governa l'ACL, e un generatore non deve prenderla al posto suo.
func TestToCoreAuthSQL_NonAssegnaNullaAiRuoli(t *testing.T) {
	script := toCoreAuthSQL([]capabilityEntry{{ID: "X", Category: "api"}})

	for _, table := range []string{"acl_role", "acl_role_group", "acl_role_cap", "acl_context", "acl_app"} {
		if strings.Contains(script, "INTO "+table+" ") || strings.Contains(script, "INTO "+table+"(") {
			t.Errorf("il seed scrive su %s: ruoli, contesti e app non gli appartengono", table)
		}
	}
}

func TestToCoreAuthSQL_EscapeDegliApici(t *testing.T) {
	script := toCoreAuthSQL([]capabilityEntry{
		{ID: "X", Category: "api", Description: "L'anagrafica"},
	})
	if !strings.Contains(script, "'L''anagrafica'") {
		t.Errorf("apice non raddoppiato nello script:\n%s", script)
	}
}

// Il documento mongo del seed deve avere la forma che mongosource smista: _et maiuscolo e
// sottodocumento api sulle sole capability di categoria "api".
func TestToCoreAuthMongo_FormaDelDocumento(t *testing.T) {
	script := toCoreAuthMongo([]capabilityEntry{
		{ID: "GET_PERSONS", Category: "api", Endpoint: "/api/persons", Method: "GET"},
		{ID: "EXPORT", Category: "action_api", Description: "Export"},
	})

	for _, want := range []string{`"_et": "CAPABILITY"`, `"_et": "CAPABILITYGROUP"`, `"operationid"`, `"acl"`} {
		if !strings.Contains(script, want) {
			t.Errorf("il seed non contiene %s:\n%s", want, script)
		}
	}
	if strings.Contains(script, "cap-def") {
		t.Error("il seed per go-core-auth non deve usare il vocabolario del frontdoor OPEM")
	}

	// Una action_api non ha rotta: un sottodocumento api lì sarebbe un campo che nessuno legge, e
	// il validator della collection lo rifiuta.
	for _, doc := range strings.Split(script, "db.getCollection(COLLECTION).replaceOne(")[1:] {
		if !strings.Contains(doc, `"action_api"`) {
			continue
		}
		if strings.Contains(doc, `"api": {`) {
			t.Errorf("sottodocumento api su una capability action_api:\n%s", doc)
		}
	}
}

// /acl.sql resta al frontdoor OPEM: è il suo destinatario storico, e riorientarlo lo toglierebbe
// a chi lo usa.
func TestToCapabilitiesSQL_RestaSulloSchemaOPEM(t *testing.T) {
	script := toCapabilitiesSQL([]capabilityEntry{{ID: "X", Category: "api"}})
	if !strings.Contains(script, "opem_acl_cap_def") {
		t.Errorf("/acl.sql non scrive più su opem_acl_cap_def:\n%s", script)
	}
}
