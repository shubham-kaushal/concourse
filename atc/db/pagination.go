package db

type Page struct {
	From int // inclusive
	To   int // inclusive

	Limit   int
	UseDate bool
}

type Pagination struct {
	Previous *Page
	Next     *Page
}

func newIntPtr(i int) *int {
	return &i
}
