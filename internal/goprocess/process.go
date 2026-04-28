package process

type RunSpec struct {
	Name         string
	Command      string
	Args         []string
	Dir          string
	Env          []string
	After        []string
	AllowFailure bool
}
