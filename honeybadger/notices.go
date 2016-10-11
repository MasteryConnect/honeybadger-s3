package honeybadger

import (
	"encoding/json"

	log "github.com/Sirupsen/logrus"
)

type Notices struct {
	FaultId    int      `json:"-"`
	ResultIdx  int      `json:"-"`
	CallNeeded bool     `json:"-"`
	Request    *Request `json:"-"`
	Results    []Notice `json:"results"`
	Links      Links    `json:"links"`
}

type Notice struct {
	Id               string                 `json:"id"`
	FaultId          int                    `json:"fault_id"`
	EnvName          string                 `json:"environment>environment_name"`
	EnvHostname      string                 `json:"environment>hostname"`
	EnvProjectRoot   string                 `json:"environment>project_root"`
	CreatedAt        string                 `json:"created_at"`
	Message          string                 `json:"message"`
	Token            string                 `json:"token"`
	Request          request                `json:"request"`
	Backtrace        []trace                `json:"backtrace"`
	ApplicationTrace []trace                `json:"application_trace"`
	WebEnv           map[string]interface{} `json:"web_environment"`
	Cookies          map[string]interface{} `json:"cookies"`
	Url              string                 `json:"url"`
}

type request struct {
	Url       string                 `json:"url"`
	Component string                 `json:"component"`
	Action    string                 `json:"action"`
	Params    map[string]interface{} `json:"params"`
	Session   map[string]interface{} `json:"session"`
	Context   map[string]interface{} `json:"context"`
	UserEmail string                 `json:"user>email"`
	UserId    string                 `json:"user>id"`
}

type trace struct {
	Number json.Number `json:"number,Number"`
	File   string      `json:"file"`
	Method string      `json:"method"`
	Column json.Number `json:"column,Number"`
}

func NewNotices(projectId, faultId int, apiKey string, createdAfter int64, rateLimit *RateLimit) *Notices {
	return &Notices{
		FaultId:    faultId,
		ResultIdx:  -1, // Increments on each call to Next()
		CallNeeded: true,
		Request:    NewRequest(projectId, HB_API_ENDPOINT, apiKey, createdAfter, rateLimit),
	}
}

// Iterates through all of the notices. This makes an API call the first time
// this function is called, and then once the end of the current page is reached
func (p *Notices) Next() (notice *Notice, more bool) {
	// Do we need to call the api to get more notices
	moreResults := p.moreResults()

	if moreResults {
		p.ResultIdx += 1
		// Get the next notice from the list of notices returned from the API call
		if p.ResultIdx == (len(p.Results) - 1) {
			p.CallNeeded = true
		}
		return &p.Results[p.ResultIdx], true
	}
	return nil, false
}

func (n *Notices) hasResults() bool {
	if len(n.Results) == 0 {
		return false
	}
	return true
}

// Are there more results. If a call is needed i.e. we've reached the end of
// the last retrieved batch, then make another call. Otherwise simply return
// true i.e. we haven't finished iterating over the last batch we got
func (n *Notices) moreResults() bool {
	if n.CallNeeded {
		n.CallNeeded = false
		if n.hasResults() {
			// Already have results so get the next batch if we have a next link
			n.Results = nil
			if url := n.GetNextUrl(); !url.Empty {
				log.Debug("Notices - Calling the next page of results")
				n.Links = Links{} // Reset links, otherwise old link data remains
				n.Request.Next(url, n)
			}
		} else {
			log.Debug("Notices - Calling the first page of results")
			// First time calling notices for this fault
			n.Request.Notices(n.FaultId, n)
		}
		return n.hasResults()
	}
	return true
}

//
// Response methods
//

func (n *Notices) SetCallNeeded(needed bool) {
	n.CallNeeded = needed
}

func (n *Notices) SetResultIdx(idx int) {
	n.ResultIdx = idx
}

func (n *Notices) GetNextUrl() *URL {
	return NewURL(n.Links.Next)
}

func (n *Notices) Count() int {
	return len(n.Results)
}
