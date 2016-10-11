package honeybadger

type Faults struct {
	ResultIdx  int      `json:"-"`
	CallNeeded bool     `json:"-"`
	Request    *Request `json:"-"`
	Results    []Fault  `json:"results"`
	Links      Links    `json:"links"`
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
	NoticesCount  int      `json:"notices_count"`
	LastNoticeAt  string   `json:"last_notice_at"`
	Tags          []string `json:"tags"`
	Id            int      `json:"id"`
	Assignee      string   `json:"assignee"`
	Url           string   `json:"url"`
	Assigneed     int      `json:"assignee>id"`
	AssigneeEmail string   `json:"assignee>email"`
	AssigneeName  string   `json:"assignee>name"`
}

func NewFaults(projectId int, apiKey string, createdAfter int64, rateLimit *RateLimit) *Faults {
	return &Faults{
		ResultIdx:  -1, // Increments on each call to Next()
		CallNeeded: true,
		Request:    NewRequest(projectId, HB_API_ENDPOINT, apiKey, createdAfter, rateLimit),
	}
}

// Iterates through all of the faults. This makes an API call the first time
// this function is called, and then once the end of the current page is reached
func (f *Faults) Next() (fault *Fault, more bool) {
	// Do we need to call the api to get more faults
	moreResults := f.moreResults()

	if moreResults {
		f.ResultIdx = f.ResultIdx + 1
		// Get the next fault from the list of faults returned from the API call
		if f.ResultIdx == (len(f.Results) - 1) {
			f.CallNeeded = true
		}
		return &f.Results[f.ResultIdx], true
	}
	return nil, false
}

func (f *Faults) hasResults() bool {
	if len(f.Results) == 0 {
		return false
	}
	return true
}

func (f *Faults) moreResults() bool {
	if f.CallNeeded {
		if f.hasResults() {
			f.Request.Next(f.GetNextUrl(), f)
		} else {
			f.Request.Faults(f)
		}
		return f.hasResults()
	}
	return true
}

//
// Response methods
//

func (f *Faults) SetCallNeeded(needed bool) {
	f.CallNeeded = needed
}

func (f *Faults) SetResultIdx(idx int) {
	f.ResultIdx = idx
}

func (f *Faults) GetNextUrl() *URL {
	return NewURL(f.Links.Next)
}

func (f *Faults) Count() int {
	return len(f.Results)
}
