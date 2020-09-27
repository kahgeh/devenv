package types

type AppType string

const (
	FrontProxy AppType = "front-proxy"
	Api        AppType = "api"
)

type DeployParameters struct {
	AppType         AppType
	AppName         string
	Path            string
	EnvironmentName string
	DomainName      string
	DomainEmail     string
}

type InitialisationParameters struct {
	DomainName      string
	DomainEmail     string
	EnvironmentName string
}

type StartParameters struct {
	HostedZoneName  string
	DomainName      string
	EnvironmentName string
}
