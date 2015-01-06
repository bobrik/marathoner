package marathoner

import (
	"encoding/json"
	"errors"
	"log"
	"math/rand"
	"net/http"
	"sort"
)

// marathonResponse is response for /v2/tasks api endpoint
type marathonResponse struct {
	Tasks marathonTasks `json:"tasks"`
}

// marathonTasks is an alias for slice of marathonTask
type marathonTasks []marathonTask

// Len return length of marathonTask slice
func (mt marathonTasks) Len() int {
	return len(mt)
}

// Less compares two marathon tasks with specified indices
func (mt marathonTasks) Less(i, j int) bool {
	return mt[i].Id < mt[j].Id
}

// Swap swaps two marathon tasks with specified indices
func (mt marathonTasks) Swap(i, j int) {
	mt[i], mt[j] = mt[j], mt[i]
}

// marathonTask is a task from /v2/tasks api endpoint
type marathonTask struct {
	App                string                          `json:"appId"`
	Id                 string                          `json:"id"`
	Host               string                          `json:"host"`
	Ports              []int                           `json:"ports"`
	ServicePorts       []int                           `json:"servicePorts"`
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
}

// NewMarathon creates new marathon client with specified endpoints
func NewMarathon(endpoints []string) Marathon {
	return Marathon{
		endpoints: endpoints,
	}
}

// State returns running and healthy marathon tasks
func (m Marathon) State() (State, error) {
	resp, err := m.fetchTasks()
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

	sort.Sort(mr.Tasks)

	for _, t := range mr.Tasks {
		if len(t.ServicePorts) == 0 || len(t.Ports) == 0 {
			continue
		}

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

		app, ok := state[t.App]
		if !ok {
			app = App{
				Name:  t.App,
				Ports: t.ServicePorts,
			}
		}

		if t.StartedAt == "" {
			continue
		}

		if t.StartedAt != "" {
			t.StartedAt = "see https://github.com/mesosphere/marathon/issues/918"
		}

		task := Task{
			Id:        t.Id,
			Host:      t.Host,
			Ports:     t.Ports,
			StagedAt:  t.StagedAt,
			StartedAt: t.StartedAt,
		}

		app.Tasks = append(app.Tasks, task)

		state[app.Name] = app
	}

	return state, nil
}

// fetchTasks fetches tasks from random alive marathon server
func (m Marathon) fetchTasks() (*http.Response, error) {
	for _, i := range rand.Perm(len(m.endpoints)) {
		resp, err := http.Get(m.endpoints[i] + "/v2/tasks")
		if err != nil {
			log.Println("error fetching marathon tasks from " + m.endpoints[i] + ", " + err.Error())
			continue
		}

		return resp, nil
	}

	return nil, errors.New("task list fetching failed on all marathon endpoints")
}
