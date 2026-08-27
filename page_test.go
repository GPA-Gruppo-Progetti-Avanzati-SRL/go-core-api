package coreapi

import (
	"errors"
	"strings"
	"testing"

	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/go-core-app/page"
)

func TestPagingRequestGetSort(t *testing.T) {
	t.Run("sort vuoto: nessun criterio, nessun errore", func(t *testing.T) {
		got, appErr := (&PagingRequest{}).GetSort()
		if appErr != nil {
			t.Fatalf("atteso nil, ottenuto %s", appErr.Message)
		}
		if got != nil {
			t.Errorf("attesa SortRequest nil, ottenuta %v", got)
		}
	})

	t.Run("multi campo, ordine preservato", func(t *testing.T) {
		got, appErr := (&PagingRequest{Sort: "name:asc,createdAt:desc"}).GetSort()
		if appErr != nil {
			t.Fatalf("atteso nil, ottenuto %s", appErr.Message)
		}
		want := page.SortRequest{{Field: "name", Dir: page.Asc}, {Field: "createdAt", Dir: page.Desc}}
		if len(got) != len(want) {
			t.Fatalf("attesi %d criteri, ottenuti %d", len(want), len(got))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("criterio %d = %+v, atteso %+v", i, got[i], want[i])
			}
		}
	})

	t.Run("direzione invalida: errore di business con la causa del parser", func(t *testing.T) {
		_, appErr := (&PagingRequest{Sort: "name:sideways"}).GetSort()
		if appErr == nil {
			t.Fatal("attesa una violazione")
		}
		if appErr.StatusCode != 422 || appErr.Code != "ERR-SORT" {
			t.Errorf("status/code = %d/%s, attesi 422/ERR-SORT", appErr.StatusCode, appErr.Code)
		}
		// La causa è l'errore di ParseSort, non la foglia sintetica code+message.
		cause := errors.Unwrap(appErr)
		if cause == nil {
			t.Fatal("la causa deve essere presente")
		}
		if !strings.Contains(cause.Error(), "invalid direction") {
			t.Errorf("causa = %q, atteso l'errore di page.ParseSort", cause.Error())
		}
	})
}
