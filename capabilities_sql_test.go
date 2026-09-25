package coreapi

import (
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/go-core-auth/sqlsource"
)

// Il seed di /acl.coreauth.sql lo scrive questo modulo, le tabelle le legge go-core-auth/sqlsource:
// sono due moduli rilasciati separatamente, e nulla li tiene allineati a compile-time. Questi test
// sono il presidio — una colonna rinominata da un lato diventa un test rosso invece di una query
// che in produzione non trova la colonna.
//
// /acl.sql e /acl.mongo.js non sono coperti da qui: hanno un altro destinatario (il frontdoor
// OPEM) e un altro schema, e il presidio di quel contratto non sta in questo repo.

var insertRe = regexp.MustCompile(`(?s)INSERT INTO (\w+) \(([^)]*)\)`)

func TestToCoreAuthSQL_ScriveSulloSchemaCheSqlsourceLegge(t *testing.T) {
	script := toCoreAuthSQL([]capabilityEntry{
		{ID: "GET_PERSONS", Category: "api", Description: "Elenco", Endpoint: "/api/persons", Method: "GET"},
		{ID: "EXPORT", Category: "action_api", Description: "Export massivo"},
	})

	schema := sqlsource.Schema()
	matches := insertRe.FindAllStringSubmatch(script, -1)
	if len(matches) == 0 {
		t.Fatalf("nessuna INSERT generata:\n%s", script)
	}

	seen := map[string]bool{}
	for _, m := range matches {
		table, cols := m[1], splitColumns(m[2])
		seen[table] = true

		known, ok := schema[table]
		if !ok {
			t.Errorf("INSERT su %q, che non è una tabella di sqlsource: %v", table, sqlsource.TableNames())
			continue
		}
		for _, c := range cols {
			if !slices.Contains(known, c) {
				t.Errorf("tabella %s: colonna %q non esiste nel modello di sqlsource (%v)", table, c, known)
			}
		}
	}

	for _, want := range []string{"acl_capability", "acl_capability_group", "acl_capability_group_item"} {
		if !seen[want] {
			t.Errorf("il seed non scrive su %s", want)
		}
	}
}

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

func splitColumns(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(strings.ReplaceAll(p, "\n", "")); t != "" {
			out = append(out, t)
		}
	}
	return out
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
