package dbtask

// BatchTask is a list of extract jobs executed in order.
type BatchTask struct {
	Tasks []ExtractTask `json:"tasks" yaml:"tasks"`
}
