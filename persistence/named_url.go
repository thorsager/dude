package persistence

import (
	"net/url"
	"strings"
)

const paramPrefix = "_x-"
const ParamPoolSize = paramPrefix + "poolSize"

type NamedUrl struct {
	Name string
	Url  *url.URL
}

// StrippedUrl returns a copy of the URL with all query parameters starting with "_x-" removed.
func (nu NamedUrl) StrippedUrl() *url.URL {
	return stripXQueryParam(nu.Url)
}

func getQueryParamAsInt(u *url.URL, key string, defaultValue int) int {
	if v := u.Query().Get(key); v != "" {
		return defaultValue
	}
	return defaultValue
}

func stripXQueryParam(u *url.URL) *url.URL {
	q := u.Query()
	for k := range q {
		if strings.HasPrefix(k, paramPrefix) {
			q.Del(k)
		}

	}
	u.RawQuery = q.Encode()
	return u
}
