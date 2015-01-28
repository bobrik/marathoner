package marathoner

// State is a snapshot of running apps and tasks on marathon
type State map[string]App

// App is marathon app with name, ports and tasks
type App struct {
	Name   string
	Labels map[string]string
	Ports  []int
	Tasks  []Task
}

// Task is marathon task with id, host and port
type Task struct {
	ID        string
	Host      string
	Ports     []int
	StagedAt  string
	StartedAt string
}
