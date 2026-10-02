package store

import "fmt"

type Page struct{ Limit, Offset int }

func NewPage(limit, offset int) (Page, error) {
	if limit == 0 {
		limit = 50
	}
	if limit < 1 || limit > 100 {
		return Page{}, fmt.Errorf("limit must be between 1 and 100")
	}
	if offset < 0 {
		return Page{}, fmt.Errorf("offset must not be negative")
	}
	return Page{Limit: limit, Offset: offset}, nil
}
