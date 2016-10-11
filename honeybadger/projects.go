package honeybadger

import (
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
)

type Projects struct {
	ProjectIncludeList map[string]bool `json:"-"`
	IncludeAll         bool            `json:"-"`
	ResultIdx          int             `json:"-"`
	CallNeeded         bool            `json:"-"`
	Request            *Request        `json:"-"`
	Results            []Project       `json:"results"`
	Links              Links           `json:"links"`
}

type Project struct {
	Id                   int       `json:"id"`
	Name                 string    `json:"name"`
	Token                string    `json:"token"`
	CreatedAt            time.Time `json:"created_at"`
	Teams                []team    `json:"teams"`
	Environments         []string  `json:"environments"`
	OwnerId              int       `json:"owner>id"`
	OwnerEmail           string    `json:"owner>email"`
	OnwerName            string    `json:"owner>name"`
	LastNoticeAt         time.Time `json:"last_notice_at"`
	EarliestNoticeAt     time.Time `json:"earliest_notice_at"`
	UnresolvedFaultCount int       `json:"unresolved_fault_count"`
	FaultCount           int       `json:"fault_count"`
	Active               bool      `json:"active"`
	Users                []user    `json:"users"`
	Sites                []site    `json:"sites"`
}

type team struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type user struct {
	Id    int    `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type site struct {
	Id            string    `json:"id"`
	Active        bool      `json:"active"`
	LastCheckedAt time.Time `json:"last_checked_at"`
	Name          string    `json:"name"`
	State         string    `json:"state"`
	Url           string    `json:"url"`
}

func NewProjects(projects, apiKey string, rateLimit *RateLimit) *Projects {
	projectIncludeList := parseProjectList(projects)
	includeAll := false
	if len(projectIncludeList) < 1 {
		includeAll = true
	}
	return &Projects{
		ProjectIncludeList: projectIncludeList,
		IncludeAll:         includeAll,
		ResultIdx:          -1, // Increments on each call to Next()
		CallNeeded:         true,
		Request:            NewRequest(0, HB_API_ENDPOINT, apiKey, 0, rateLimit),
	}
}

// Iterates through all of the projects. This makes an API call the first time
// this function is called, and then once the end of the current page is reached
func (p *Projects) Next() (project *Project, more bool) {
	if p.IncludeAll {
		return p.includeAllNext()
	} else {
		return p.includeListNext()
	}
}

func (p *Projects) includeAllNext() (project *Project, more bool) {
	// Do we need to call the api to get more projects
	moreResults := p.moreResults()

	if moreResults {
		p.ResultIdx = p.ResultIdx + 1
		// Get the next project from the list of projects returned from the API call
		if p.ResultIdx == (len(p.Results) - 1) {
			p.CallNeeded = true
		}
		return &p.Results[p.ResultIdx], true
	}
	return nil, false
}

func (p *Projects) includeListNext() (project *Project, more bool) {
	// Do we need to call the api to get more projects
	moreResults := p.moreResults()

	for moreResults {
		// Get the next project from the list of projects returned from the API call
		found := false
		for i := p.ResultIdx + 1; i < len(p.Results) && !found; i++ {
			if i == (len(p.Results) - 1) {
				p.CallNeeded = true
			}
			project = &p.Results[i]
			if p.ProjectIncludeList[strings.ToLower(project.Name)] {
				p.ResultIdx = i
				found = true
			}
		}
		// Found a project, either because we're including all projects, or it
		// matches one in the include list
		if found {
			return &p.Results[p.ResultIdx], true
		} else {
			// We've checked all projects in the current page of results and didn't
			// find a project that is part of the include list. Check the next page
			moreResults = p.moreResults()
		}
	}
	// We've checked all pages and there are no more results
	return nil, false
}

func (p *Projects) hasResults() bool {
	if len(p.Results) == 0 {
		return false
	}
	return true
}

func (p *Projects) moreResults() bool {
	if p.CallNeeded {
		if p.hasResults() {
			p.Request.Next(p.GetNextUrl(), p)
		} else {
			p.Request.Projects(p)
		}
		return p.hasResults()
	}
	return true
}

func parseProjectList(projects string) map[string]bool {
	projectsList := strings.Split(projects, ",")
	projectsHash := make(map[string]bool)
	for _, v := range projectsList {
		p := strings.TrimSpace(v)
		if len(p) > 0 {
			projectsHash[strings.ToLower(p)] = true
		}
	}
	log.WithFields(log.Fields{"projects": projectsHash}).Info("Project List: ")
	return projectsHash
}

//
// Response methods
//

func (p *Projects) SetCallNeeded(needed bool) {
	p.CallNeeded = needed
}

func (p *Projects) SetResultIdx(idx int) {
	p.ResultIdx = idx
}

func (p *Projects) GetNextUrl() *URL {
	return NewURL(p.Links.Next)
}

func (p *Projects) Count() int {
	return len(p.Results)
}
