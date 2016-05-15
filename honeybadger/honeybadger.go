package honeybadger

import (
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	HB_API_ENDPOINT = "https://api.honeybadger.io/v1/projects"
	HTTP_TIMEOUT    = time.Duration(5 * time.Minute)
)

type Response interface {
	SetCallNeeded(needed bool)
	SetResultIdx(idx int)
}

type URL struct {
	Url        *url.URL
	PathParams []string
	Values     url.Values
}

func CallHB(hbUrl string, results Response) {
	client := &http.Client{Timeout: HTTP_TIMEOUT}
	req, err := http.NewRequest("GET", hbUrl, nil)
	if err != nil {
		log.Fatalln(err)
	}
	req.Header.Add("Accept", "application/json")
	retryCount := 0
	for retryCount < 5 {
		resp, err := client.Do(req)
		if err != nil {
			if retryCount < 5 {
				retryCount++
				continue // try again
			} else {
				log.WithFields(log.Fields{"retries": retryCount}).Info("Client call failed: ")
				log.Fatalln(err)
			}
		}
		defer resp.Body.Close()
		decoder := json.NewDecoder(resp.Body)
		err = decoder.Decode(results)
		if err != nil {
			log.Fatalln(err)
		}
		results.SetCallNeeded(false)
		results.SetResultIdx(-1)
		break
	}

}

func NewURL(u string) *URL {
	newUrl, err := url.Parse(u)
	if err != nil {
		log.Fatalln(err)
	}
	return &URL{Url: newUrl, Values: newUrl.Query()}
}

// Mutates the URL to set the API Key query param
func (u *URL) SetApiKey(key string) *URL {
	u.Values.Set("auth_token", key)
	return u
}

// Mutates the URL to set the page query param
func (u *URL) SetPage(page int) *URL {
	u.Values.Set("page", strconv.Itoa(page))
	return u
}

// Mutates the URL to set the limit query param
func (u *URL) SetLimit(limit int) *URL {
	u.Values.Set("limit", strconv.Itoa(limit))
	return u
}

// Mutates the URL to set the occurred_after query param
func (u *URL) SetOccurredAfter(timestamp int64) *URL {
	u.Values.Set("occurred_after", strconv.FormatInt(timestamp, 10))
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

// Adds the project faults path to the URL.PathParams. Not realized in the URL
// until URL.String() is called
func (u *URL) ProjectFaults(projectId int) string {
	urlParts := strings.Split(u.String(), "?")
	urlParts = Insert(urlParts, 1, "/", strconv.Itoa(projectId), "/", "faults", "?")
	return strings.Join(urlParts, "")
}

// Adds the notices path to the URL.PathParams. Not realized in the URL
// until URL.String() is called
func (u *URL) FaultNotices(projectId, faultId int) string {
	urlParts := strings.Split(u.String(), "?")
	urlParts = Insert(urlParts, 1, "/", strconv.Itoa(projectId), "/", "faults", "/", strconv.Itoa(faultId), "/", "notices", "?")
	return strings.Join(urlParts, "")
}

func (u *URL) ResetPathParams() {
	u.PathParams = []string{}
}

// Insert inserts the values into the slice at the specified index, which must
// be in range. The slice must have room for the new element.
func Insert(slice []string, index int, values ...string) []string {
	return append(slice[:index], append(values, slice[index:]...)...)
}
