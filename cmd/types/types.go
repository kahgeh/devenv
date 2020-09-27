package types

import "fmt"

type ArgName string

const (
	ArgEnvName     ArgName = "env-name"
	ArgDomainName  ArgName = "domain-name"
	ArgDomainEmail ArgName = "domain-email"
)

type KnownApp string

const (
	KnownAppFrontProxy KnownApp = "front-proxy"
)

type MissingArgument struct {
	ParameterName string
}

func (e *MissingArgument) Error() string {
	return fmt.Sprintf("%q is required", e.ParameterName)
}
