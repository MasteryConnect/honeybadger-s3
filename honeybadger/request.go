package honeybadger

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
)

type Request struct {
	Endpoint     string
	ApiKey       string
	CreatedAfter int64
	ProjectId    int
	RateLimit    *RateLimit
}

type RateLimit struct {
	Limit     int
	Remaining int
	Reset     int64
}

func NewRequest(projectId int, endpoint, apiKey string, createdAfter int64, rateLimit *RateLimit) *Request {
	return &Request{
		Endpoint:     endpoint,
		ApiKey:       apiKey,
		ProjectId:    projectId,
		CreatedAfter: createdAfter,
		RateLimit:    rateLimit,
	}
}

func (r *Request) Projects(results Response) {
	urlStr := NewURL(r.Endpoint).String()
	log.WithFields(log.Fields{
		"url": urlStr,
	}).Debug("run data")
	r.CallHB(urlStr, results)
}

func (r *Request) Faults(results Response) {
	var url *URL
	if r.CreatedAfter > 0 {
		url = NewURL(r.Endpoint).SetCreatedAfter(r.CreatedAfter)
	} else {
		url = NewURL(r.Endpoint)
	}
	urlParts := strings.Split(url.String(), "?")
	urlParts = Insert(urlParts, 1, "/", strconv.Itoa(r.ProjectId), "/", "faults", "?")
	hbUrl := strings.Join(urlParts, "")
	log.WithFields(log.Fields{
		"url": hbUrl,
	}).Debug("run data")
	r.CallHB(hbUrl, results)
}

func (r *Request) Notices(faultId int, results Response) {
	var url *URL
	if r.CreatedAfter > 0 {
		url = NewURL(r.Endpoint).SetCreatedAfter(r.CreatedAfter)
	} else {
		url = NewURL(r.Endpoint)
	}
	urlParts := strings.Split(url.String(), "?")
	urlParts = Insert(urlParts, 1, "/", strconv.Itoa(r.ProjectId), "/", "faults", "/", strconv.Itoa(faultId), "/", "notices", "?")
	hbUrl := strings.Join(urlParts, "")
	log.WithFields(log.Fields{
		"url": hbUrl,
	}).Debug("run data")
	r.CallHB(hbUrl, results)
}

// The URL is fully formed (except created_after which may be a bug), simply
// pass on to CallHB. This can be removed if we don't have to add created_after
// i.e. it is a bug that HB fixes
func (r *Request) Next(url *URL, results Response) {
	// Seems like a bug that we have to add created after to next
	if r.CreatedAfter > 0 {
		url = url.SetCreatedAfter(r.CreatedAfter)
	}
	urlStr := url.String()
	log.WithFields(log.Fields{
		"url": urlStr,
	}).Debug("run data")
	r.CallHB(urlStr, results)
}

func (r *Request) CallHB(url string, results Response) {
	if r.RateLimit.Limit > 0 && r.RateLimit.Remaining == 0 {
		resetTime := time.Unix(r.RateLimit.Reset, 0)
		log.WithFields(log.Fields{
			"limit":     r.RateLimit.Limit,
			"remaining": r.RateLimit.Remaining,
			"reset":     resetTime,
		}).Warn("Waiting - we have reached the rate limit:")
		log.WithFields(log.Fields{
			"until": resetTime,
		}).Warn("Waiting")
		time.Sleep(resetTime.Sub(time.Now()))
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatalln(err)
	}
	req.Header.Add("Accept", "application/json")
	req.SetBasicAuth(r.ApiKey, "")
	retryCount := 0
	client := &http.Client{Timeout: HTTP_TIMEOUT}
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

		headers := resp.Header
		r.RateLimit.Limit, _ = strconv.Atoi(headers.Get("X-RateLimit-Limit"))
		r.RateLimit.Remaining, _ = strconv.Atoi(headers.Get("X-RateLimit-Remaining"))
		r.RateLimit.Reset, _ = strconv.ParseInt(headers.Get("X-RateLimit-Reset"), 10, 64)
		log.WithFields(log.Fields{
			"limit":     r.RateLimit.Limit,
			"remaining": r.RateLimit.Remaining,
			"reset":     r.RateLimit.Reset,
		}).Debug("Rate Limiting:")

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
