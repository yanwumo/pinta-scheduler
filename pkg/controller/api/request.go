package api

type Request struct {
	Namespace string
	JobName   string
	TaskName  string
	QueueName string
}
