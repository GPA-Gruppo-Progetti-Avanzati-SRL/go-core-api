package coreapi

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeBusiness interface{ Do() }

type fakeBusinessImpl struct{}

func (fakeBusinessImpl) Do() {}

type capturedInvoke struct {
	fn    any
	modes []string
}

// stubInvoke sostituisce la var di package invoke per la durata del test, così si può
// ispezionare cosa Module accoda a fx senza far girare un'app fx.
func stubInvoke(t *testing.T) *[]capturedInvoke {
	t.Helper()
	prev := invoke
	captured := make([]capturedInvoke, 0, 2)
	invoke = func(fn any, modes ...string) {
		captured = append(captured, capturedInvoke{fn: fn, modes: modes})
	}
	t.Cleanup(func() { invoke = prev })
	return &captured
}

func TestWithRoutesAnyOptionOrder(t *testing.T) {
	register := func(r *Router, b fakeBusiness) {}

	t.Run("routes before modes", func(t *testing.T) {
		captured := stubInvoke(t)
		Module(&Config{}, WithRoutes(register), WithModes("api"))
		require.Len(t, *captured, 1)
		assert.Equal(t, []string{"api"}, (*captured)[0].modes)
	})

	t.Run("modes before routes", func(t *testing.T) {
		captured := stubInvoke(t)
		Module(&Config{}, WithModes("api"), WithRoutes(register))
		require.Len(t, *captured, 1)
		assert.Equal(t, []string{"api"}, (*captured)[0].modes)
	})
}

func TestWithRoutesSignature(t *testing.T) {
	captured := stubInvoke(t)

	Module(&Config{}, WithRoutes(func(r *Router, b fakeBusiness) {}))

	require.Len(t, *captured, 1)
	// La funzione data a fx deve chiedere *Router e il tipo del business inferito da B:
	// è così che dig risolve il business per tipo e forza la costruzione del Router.
	want := reflect.TypeOf(func(*Router, fakeBusiness) {})
	assert.Equal(t, want, reflect.TypeOf((*captured)[0].fn))
}

func TestWithRoutesDispatchesToRegister(t *testing.T) {
	captured := stubInvoke(t)

	calls := 0
	var gotRouter *Router
	var gotBusiness fakeBusiness
	Module(&Config{}, WithRoutes(func(r *Router, b fakeBusiness) {
		calls++
		gotRouter, gotBusiness = r, b
	}))

	require.Len(t, *captured, 1)
	fn, ok := (*captured)[0].fn.(func(*Router, fakeBusiness))
	require.True(t, ok)

	router := &Router{}
	business := fakeBusinessImpl{}
	fn(router, business)

	assert.Equal(t, 1, calls)
	assert.Same(t, router, gotRouter)
	assert.Equal(t, business, gotBusiness)
}

func TestWithRoutesOnePerRegistration(t *testing.T) {
	captured := stubInvoke(t)

	Module(&Config{},
		WithRoutes(func(r *Router, b fakeBusiness) {}),
		WithRoutes(func(r *Router, b *fakeBusinessImpl) {}),
	)

	assert.Len(t, *captured, 2)
}

func TestModuleWithoutRoutesRegistersNoInvoke(t *testing.T) {
	captured := stubInvoke(t)

	Module(&Config{}, WithModes("api"))

	assert.Empty(t, *captured)
}
