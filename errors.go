package africastalkinggo

type ApiError struct {
	Status   string
	Endpoint string
	Message  string
	Body     string
}
