package honeybadger

import (
	"encoding/json"
	log "github.com/Sirupsen/logrus"
)

type Notices struct {
	ProjectId     int      `json:"-"`
	FaultId       int      `json:"-"`
	ResultIdx     int      `json:"-"`
	CallNeeded    bool     `json:"-"`
	ApiKey        string   `json:"-"`
	OccurredAfter int64    `json:"-"`
	Results       []Notice `json:"results"`
	TotalCount    int      `json:"total_count"`
	CurrentPage   int      `json:"current_page"`
	NumPages      int      `json:"num_pages"`
}

type Notice struct {
	Id               int                    `json:"id"`
	FaultId          int                    `json:"fault_id"`
	Environment      Environment            `json:"environment"`
	CreatedAt        string                 `json:"created_at"`
	Message          string                 `json:"message"`
	Token            string                 `json:"token"`
	Request          Request                `json:"request"`
	Backtrace        []Trace                `json:"backtrace"`
	ApplicationTrace []Trace                `json:"application_trace"`
	WebEnv           map[string]interface{} `json:"web_environment"`
	Deploy           Deploy                 `json:"deploy"`
	Url              string                 `json:"url"`
}

type Request struct {
	Url       string                 `json:"url"`
	Component string                 `json:"component"`
	Action    string                 `json:"action"`
	Params    map[string]interface{} `json:"params"`
	Session   map[string]interface{} `json:"session"`
	Context   map[string]interface{} `json:"context"`
}

type Deploy struct {
	Environment   string `json:"environment"`
	Revision      string `json:"revision"`
	Repository    string `json:"repository"`
	LocalUsername string `json:"local_username"`
	CreatedAt     string `json:"created_at"`
	Url           string `json:"url"`
}

type Trace struct {
	Number json.Number `json:"number,Number"`
	File   string      `json:"file"`
	Method string      `json:"method"`
}

func NewNotices(projectId, faultId int, apiKey string, occurredAfter int64) *Notices {
	return &Notices{
		ProjectId:     projectId,
		FaultId:       faultId,
		ResultIdx:     -1, // Increments on each call to Next()
		CurrentPage:   -1, // So the first next page call passes
		CallNeeded:    true,
		ApiKey:        apiKey,
		OccurredAfter: occurredAfter,
	}
}

// Loads the Notices struct with the notices on the given page argument
func (p *Notices) GetNotices(page int) {
	var hbUrl string
	if p.OccurredAfter > 0 {
		hbUrl = NewURL(HB_API_ENDPOINT).SetApiKey(p.ApiKey).SetPage(page).SetCreatedAfter(p.OccurredAfter).FaultNotices(p.ProjectId, p.FaultId)
	} else {
		hbUrl = NewURL(HB_API_ENDPOINT).SetApiKey(p.ApiKey).SetPage(page).FaultNotices(p.ProjectId, p.FaultId)
	}
	log.WithFields(log.Fields{
		"url": hbUrl,
	}).Debug("run data")
	CallHB(hbUrl, p)
}

// Iterates through all of the notices. This makes an API call the first time
// this function is called, and then once the end of the current page is reached
func (p *Notices) Next() (notice *Notice, more bool) {
	// Do we need to call the api to get more notices
	moreResults := p.moreResults()

	if moreResults {
		p.ResultIdx = p.ResultIdx + 1
		// Get the next notice from the list of notices returned from the API call
		if p.ResultIdx == (len(p.Results) - 1) {
			p.CallNeeded = true
		}
		return &p.Results[p.ResultIdx], true
	}
	return nil, false
}

func (f *Notices) hasResults() bool {
	if f.TotalCount == 0 {
		return false
	}
	return true
}

func (f *Notices) moreResults() bool {
	if f.CallNeeded {
		if nextPage, morePages := f.NextPage(); morePages {
			f.GetNotices(nextPage)
			return f.hasResults()
		} else {
			return false
		}
	}
	return true
}

// Returns the page number for the next page and true if there are more pages.
// If no more pages are available i.e. Notices.CurrentPage == Notices.NumPages,
// then -1 and false is returned
func (p *Notices) NextPage() (nextPage int, morePages bool) {
	if p.CurrentPage < p.NumPages {
		return p.CurrentPage + 1, true
	} else {
		return -1, false
	}
}

//
// Response methods
//

func (p *Notices) SetCallNeeded(needed bool) {
	p.CallNeeded = needed
}

func (p *Notices) SetResultIdx(idx int) {
	p.ResultIdx = idx
}
