package apiservices

import "github.com/GPA-Gruppo-Progetti-Avanzati-SRL/go-core-app/page"

type PagedResponse[T any] struct {
	PageSize    int   `header:"pageSize"`
	TotalItems  int64 `header:"totalItems"`
	TotalPages  int   `header:"totalPages"`
	CurrentPage int   `header:"currentPage"`
	HasNext     bool  `header:"hasNext"`
	HasPrev     bool  `header:"hasPrev"`
	Body        []T
}
type PagingRequest struct {
	PageSize   int `query:"pagesize" default:"-1"`
	PageNumber int `query:"pagenumber" default:"-1"`
}

func GeneratePageResponse[T any](body []T, paging *page.Paging) *PagedResponse[T] {

	return &PagedResponse[T]{
		PageSize:    paging.PageSize,
		TotalItems:  paging.TotalItems,
		TotalPages:  paging.TotalPages,
		CurrentPage: paging.CurrentPage,
		HasNext:     paging.HasNext,
		HasPrev:     paging.HasPrev,
		Body:        body,
	}
}
