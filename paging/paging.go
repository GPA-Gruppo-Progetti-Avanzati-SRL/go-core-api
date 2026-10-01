// Package paging contiene i tipi di richiesta e risposta paginata delle operazioni huma: i parametri
// in query (PagingRequest) e gli header della pagina (PagedResponse), costruiti da page.Paging.
package paging

import (
	apierrors "github.com/GPA-Gruppo-Progetti-Avanzati-SRL/go-core-api/internal/errors"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/go-core-app"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/go-core-app/page"
)

type PagedResponse[T any] struct {
	PageSize    int   `header:"pageSize"`
	TotalCount  int64 `header:"totalCount"`
	TotalPages  int   `header:"totalPages"`
	CurrentPage int   `header:"currentPage"`
	HasNext     bool  `header:"hasNext"`
	HasPrevious bool  `header:"hasPrevious"`
	Body        []T
}

type PagingRequest struct {
	PageSize   int    `query:"pagesize" default:"-1"`
	PageNumber int    `query:"pagenumber" default:"-1"`
	Sort       string `query:"sort"` // es. "name:asc,createdAt:desc"
}

// GetSort parses the Sort query param into a SortRequest.
// Returns nil without error when Sort is empty.
func (p *PagingRequest) GetSort() (page.SortRequest, *core.Error) {
	s, err := page.ParseSort(p.Sort)
	if err != nil {
		return nil, core.BusinessError().WithAmbit(apierrors.Ambit).WithCode(apierrors.CodeSort).WithCause(err)
	}
	return s, nil
}

func GeneratePageResponse[T any](body []T, paging *page.Paging) *PagedResponse[T] {

	return &PagedResponse[T]{
		PageSize:    paging.PageSize,
		TotalCount:  paging.TotalCount,
		TotalPages:  paging.TotalPages,
		CurrentPage: paging.CurrentPage,
		HasNext:     paging.HasNext,
		HasPrevious: paging.HasPrevious,
		Body:        body,
	}
}
