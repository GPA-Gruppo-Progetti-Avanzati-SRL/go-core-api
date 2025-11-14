package authorization

// RoleMatcher definisce la logica di matching tra i ruoli disponibili e l'identificatore richiesto.
// Implementazioni diverse possono essere iniettate via Router.SetRoleMatcher.
type RoleMatcher interface {
	Match(roles []string, required string) bool
}
