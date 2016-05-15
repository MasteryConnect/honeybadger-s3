package honeybadger

import (
	log "github.com/Sirupsen/logrus"
)

type Faults struct {
	ProjectId     int     `json:"-"`
	ResultIdx     int     `json:"-"`
	CallNeeded    bool    `json:"-"`
	ApiKey        string  `json:"-"`
	OccurredAfter int64   `json:"-"`
	Results       []Fault `json:"results"`
	TotalCount    int     `json:"total_count"`
	CurrentPage   int     `json:"current_page"`
	NumPages      int     `json:"num_pages"`
}

type Fault struct {
	ProjectId     int      `json:"project_id"`
	Klass         string   `json:"klass"`
	Component     string   `json:"component"`
	Action        string   `json:"action"`
	Environment   string   `json:"environment"`
	Resolved      bool     `json:"resolved"`
	Ignored       bool     `json:"ignored"`
	CreatedAt     string   `json:"created_at"`
	CommentsCount int      `json:"comments_count"`
	Message       string   `json:"message"`
	LastNoticeAt  string   `json:"last_notice_at"`
	Tags          []string `json:"tags"`
	Id            int      `json:"id"`
	Assignee      string   `json:"assignee"`
	Tickets       []string `json:"tickets"`
}

func NewFaults(projectId int, apiKey string, occurredAfter int64) *Faults {
	return &Faults{
		ProjectId:     projectId,
		ResultIdx:     -1, // Increments on each call to Next()
		CurrentPage:   -1, // So the first next page call passes
		CallNeeded:    true,
		ApiKey:        apiKey,
		OccurredAfter: occurredAfter,
	}
}

// Loads the Faults struct with the faults on the given page argument
func (p *Faults) GetFaults(page int) {
	var hbUrl string
	if p.OccurredAfter > 0 {
		hbUrl = NewURL(HB_API_ENDPOINT).SetApiKey(p.ApiKey).SetPage(page).SetOccurredAfter(p.OccurredAfter).ProjectFaults(p.ProjectId)
	} else {
		hbUrl = NewURL(HB_API_ENDPOINT).SetApiKey(p.ApiKey).SetPage(page).ProjectFaults(p.ProjectId)
	}
	log.WithFields(log.Fields{
		"url": hbUrl,
	}).Debug("run data")
	CallHB(hbUrl, p)

}

// Iterates through all of the faults. This makes an API call the first time
// this function is called, and then once the end of the current page is reached
func (p *Faults) Next() (fault *Fault, more bool) {
	// Do we need to call the api to get more faults
	moreResults := p.moreResults()

	if moreResults {
		p.ResultIdx = p.ResultIdx + 1
		// Get the next fault from the list of faults returned from the API call
		if p.ResultIdx == (len(p.Results) - 1) {
			p.CallNeeded = true
		}
		return &p.Results[p.ResultIdx], true
	}
	return nil, false
}

func (f *Faults) hasResults() bool {
	if f.TotalCount == 0 {
		return false
	}
	return true
}

func (f *Faults) moreResults() bool {
	if f.CallNeeded {
		if nextPage, morePages := f.NextPage(); morePages {
			f.GetFaults(nextPage)
			return f.hasResults()
		} else {
			return false
		}
	}
	return true
}

// Returns the page number for the next page and true if there are more pages.
// If no more pages are available i.e. Faults.CurrentPage == Faults.NumPages,
// then -1 and false is returned
func (p *Faults) NextPage() (nextPage int, morePages bool) {
	if p.CurrentPage < p.NumPages {
		return p.CurrentPage + 1, true
	} else {
		return -1, false
	}
}

//
// Response methods
//

func (p *Faults) SetCallNeeded(needed bool) {
	p.CallNeeded = needed
}

func (p *Faults) SetResultIdx(idx int) {
	p.ResultIdx = idx
}
