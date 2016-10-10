package honeybadger

import "time"

const (
	HB_API_ENDPOINT = "https://api.honeybadger.io/v2/projects"
	HTTP_TIMEOUT    = time.Duration(5 * time.Minute)
)

type Links struct {
	Prev string `json:"prev"`
	Self string `json:"self"`
	Next string `json:"next"`
}

type Response interface {
	SetCallNeeded(needed bool)
	SetResultIdx(idx int)
	GetNextUrl() *URL
	Count() int
}

// Insert inserts the values into the slice at the specified index, which must
// be in range. The slice must have room for the new element.
func Insert(slice []string, index int, values ...string) []string {
	return append(slice[:index], append(values, slice[index:]...)...)
}
