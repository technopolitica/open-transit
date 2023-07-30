package types

type PaginationLinks struct {
	First string `json:"first,omitempty"`
	Last  string `json:"last,omitempty"`
	Prev  string `json:"prev,omitempty"`
	Next  string `json:"next,omitempty"`
}

type PaginatedResponse struct {
	Version string          `json:"version"`
	Links   PaginationLinks `json:"links"`
}
