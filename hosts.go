package africastalking

type hosts struct {
	api      string
	voice    string
	content  string
	payments string
}

var productionHosts = hosts{
	api:      "https://api.africastalking.com",
	voice:    "https://voice.africastalking.com",
	payments: "https://payments.africastalking.com",
	content:  "https://content.africastalking.com",
}

var sandboxHosts = hosts{
	api: "https://api.sandbox.africastalking.com",
}
