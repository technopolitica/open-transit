package domain

import (
	"encoding/json"
	"net/url"
)

type URL struct {
	*url.URL
}

func (u URL) MarshalJSON() (text []byte, err error) {
	return json.Marshal(u.String())
}

func (u *URL) UnmarshalJSON(text []byte) (err error) {
	*u, err = ParseURL(string(text))
	return
}

func (u *URL) String() string {
	if u.URL == nil {
		return ""
	}
	return u.URL.String()
}

func (u *URL) ModifyQuery(mod func(query *url.Values)) URL {
	newURL := u.Clone()
	query := newURL.Query()
	mod(&query)
	newURL.RawQuery = query.Encode()
	return newURL
}

func (u *URL) Clone() URL {
	inner := *u.URL
	return URL{&inner}
}

func ParseURL(text string) (u URL, err error) {
	p, err := url.Parse(text)
	u = URL{p}
	return
}
