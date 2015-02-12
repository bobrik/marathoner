package marathoner

import (
	"encoding/json"
	"errors"
	"log"
	"math/rand"
	"net/http"
	"sort"
	"time"
)

// marathonResponse is response for /v2/apps?embed=apps.tasks api endpoint
type marathonResponse struct {
	Apps marathonApps `json:"apps"`
}

// marathonApps is an alias for slice of marathonApp
type marathonApps []marathonApp

func (ma marathonApps) Len() int {
	return len(ma)
}

func (ma marathonApps) Less(i, j int) bool {
	return ma[i].ID < ma[j].ID
}

func (ma marathonApps) Swap(i, j int) {
	ma[i], ma[j] = ma[j], ma[i]
}

// marathonApp is an app from /v2/apps?embed=apps.tasks api endpoint
type marathonApp struct {
	ID     string            `json:"id"`
	Labels map[string]string `json:"labels"`
	Ports  []int             `json:"ports"`
	Tasks  marathonTasks     `json:"tasks"`
}

// marathonTasks is an alias for slice of marathonTask
type marathonTasks []marathonTask

func (mt marathonTasks) Len() int {
	return len(mt)
}

func (mt marathonTasks) Less(i, j int) bool {
	return mt[i].ID < mt[j].ID
}

func (mt marathonTasks) Swap(i, j int) {
	mt[i], mt[j] = mt[j], mt[i]
}

// marathonTask is an embedded task from /v2/apps?embed=apps.tasks api endpoint
type marathonTask struct {
	ID                 string                          `json:"id"`
	Host               string                          `json:"host"`
	Ports              []int                           `json:"ports"`
	StagedAt           string                          `json:"stagedAt"`
	StartedAt          string                          `json:"startedAt"`
	HealthCheckResults []marathonTaskHealthCheckResult `json:"healthCheckResults"`
}

// marathonTaskHealthCheckResult is a health check result for a task
type marathonTaskHealthCheckResult struct {
	Alive bool `json:"alive"`
}

// Marathon is marathon api client
type Marathon struct {
	endpoints []string
	rand      *rand.Rand
}

// NewMarathon creates new marathon client with specified endpoints
func NewMarathon(endpoints []string) Marathon {
	return Marathon{
		endpoints: endpoints,
		rand:      rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// State returns running and healthy marathon tasks
func (m Marathon) State() (State, error) {
	resp, err := m.fetchApps()
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	mr := &marathonResponse{}

	d := json.NewDecoder(resp.Body)
	err = d.Decode(mr)
	if err != nil {
		return nil, err
	}

	state := map[string]App{}

	sort.Sort(mr.Apps)

	for _, a := range mr.Apps {
		if len(a.Ports) == 0 {
			continue
		}

		// servicePort for docker can be set, but ports still can be [0]
		// it's better to skip such apps for now
		foundEmptyPort := false
		for _, p := range a.Ports {
			if p == 0 {
				foundEmptyPort = true
				break
			}
		}

		if foundEmptyPort {
			continue
		}

		app, ok := state[a.ID]
		if !ok {
			app = App{
				Name:   a.ID,
				Labels: a.Labels,
				Ports:  a.Ports,
				Tasks:  []Task{},
			}
		}

		sort.Sort(a.Tasks)

		for _, t := range a.Tasks {
			alive := true
			for _, h := range t.HealthCheckResults {
				if !h.Alive {
					alive = false
					break
				}
			}

			if !alive {
				continue
			}

			if t.StartedAt == "" {
				continue
			}

			task := Task{
				ID:        t.ID,
				Host:      t.Host,
				Ports:     t.Ports,
				StagedAt:  t.StagedAt,
				StartedAt: t.StartedAt,
			}

			app.Tasks = append(app.Tasks, task)
		}

		if len(app.Tasks) == 0 {
			continue
		}

		state[app.Name] = app
	}

	return state, nil
}

// fetchApps fetches apps from random alive marathon server
func (m Marathon) fetchApps() (*http.Response, error) {
	for _, i := range m.rand.Perm(len(m.endpoints)) {
		resp, err := http.Get(m.endpoints[i] + "/v2/apps?embed=apps.tasks")
		if err != nil {
			log.Println("error fetching marathon apps from " + m.endpoints[i] + ", " + err.Error())
			continue
		}

		return resp, nil
	}

	return nil, errors.New("app list fetching failed on all marathon endpoints")
}
