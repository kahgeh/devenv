package provider

import (
	"fmt"
	"github.com/kahgeh/devenv/provider/types"

	daws "github.com/kahgeh/devenv/provider/aws"
)

type Provider string

const (
	Aws Provider = "aws"
)

// Session is the cloud provider session
type Session interface {
	Initialise(parameters *types.InitialisationParameters)
	Delete()
	Start(config *types.StartParameters)
	Stop()
	Deploy(parameters *types.DeployParameters)
}

// NotSupported error
type NotSupported struct {
	s    string
	name string
}

func (e *NotSupported) Error() string {
	return e.s
}

func NewSession(provider Provider) (Session, error) {
	switch provider {
	case Aws:
		return daws.CreateAwsSession()
	default:
		return nil, &NotSupported{s: fmt.Sprintf("Provider %v not supported", provider), name: string(provider)}
	}
}
