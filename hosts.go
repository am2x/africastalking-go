package africastalking

type hosts struct {
	api      string
	voice    string
	payments string
}

var productionHosts = hosts{
	api:      "https://api.africastalking.com/version1",
	voice:    "https://voice.africastalking.com",
	payments: "https://payments.africastalking.com",
}

var sandboxHosts = hosts{
	api: "https://api.sandbox.africastalking.com/version1",
}
