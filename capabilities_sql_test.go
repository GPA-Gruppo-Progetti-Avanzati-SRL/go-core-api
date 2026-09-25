package coreapi

import (
	"strings"
	"testing"
)

// /acl.sql e /acl.mongo.js appartengono al frontdoor OPEM. I seed per go-core-auth non stanno più
// qui: li serializza go-core-auth/apiauth, che è il modulo a cui quello schema appartiene, e questo
// modulo gli passa soltanto le capability dedotte dal registry huma (Capabilities/CapabilityID).

// Il seed descrive ciò che l'applicazione espone; assegnare le capability a un ruolo è una
// decisione di chi governa l'ACL, e un generatore non deve prenderla al posto suo.
func TestToCapabilitiesSQL_NonAssegnaNullaAiRuoli(t *testing.T) {
	script := toCapabilitiesSQL([]CapabilityEntry{{ID: "X", Category: "api"}})

	for _, table := range []string{"opem_acl_role_caps", "opem_user_role", "opem_user"} {
		if strings.Contains(script, "INTO "+table+" ") || strings.Contains(script, "INTO "+table+"(") {
			t.Errorf("il seed scrive su %s: l'assegnazione ai ruoli non gli appartiene", table)
		}
	}
}

func TestToCapabilitiesSQL_EscapeDegliApici(t *testing.T) {
	script := toCapabilitiesSQL([]CapabilityEntry{
		{ID: "X", Category: "api", Description: "L'anagrafica"},
	})
	if !strings.Contains(script, "'L''anagrafica'") {
		t.Errorf("apice non raddoppiato nello script:\n%s", script)
	}
}

// /acl.sql resta al frontdoor OPEM: è il suo destinatario storico, e riorientarlo lo toglierebbe
// a chi lo usa.
func TestToCapabilitiesSQL_RestaSulloSchemaOPEM(t *testing.T) {
	script := toCapabilitiesSQL([]CapabilityEntry{{ID: "X", Category: "api"}})
	if !strings.Contains(script, "opem_acl_cap_def") {
		t.Errorf("/acl.sql non scrive più su opem_acl_cap_def:\n%s", script)
	}
}
