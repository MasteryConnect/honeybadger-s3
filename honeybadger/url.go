package honeybadger

import (
	"log"
	"net/url"
	"strconv"
	"strings"
)

type URL struct {
	Url        *url.URL
	PathParams []string
	Values     url.Values
	Empty      bool
}

func NewURL(u string) *URL {
	if len(u) == 0 {
		return &URL{Empty: true}
	} else {
		newUrl, err := url.Parse(u)
		if err != nil {
			log.Fatalln(err)
		}
		return &URL{Url: newUrl, Values: newUrl.Query()}
	}
}

// Mutates the URL to set the limit query param
func (u *URL) SetLimit(limit int) *URL {
	u.Values.Set("limit", strconv.Itoa(limit))
	return u
}

// Mutates the URL to set the created_after query param
func (u *URL) SetCreatedAfter(timestamp int64) *URL {
	u.Values.Set("created_after", strconv.FormatInt(timestamp, 10))
	return u
}

func (u *URL) String() string {
	nu := *u
	nu.Url.RawQuery = nu.Values.Encode()
	nu.Url.Path = nu.Url.Path + strings.Join(nu.PathParams, "/")
	return nu.Url.String()
}

func (u *URL) ResetPathParams() {
	u.PathParams = []string{}
}
