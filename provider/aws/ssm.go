package aws

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/kahgeh/devenv/utils/ctx"
)

type SsmSession struct {
	api *ssm.Client
}

func NewSession() (*Session, error) {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		panic("unable to load SDK config, " + err.Error())
	}
	session := &Session{config: cfg}
	return session, nil
}

func (session *Session) NewSsmSession() *SsmSession {
	api := ssm.New(session.config)
	return &SsmSession{
		api: api,
	}
}

func (session *SsmSession) GetParamterValue(name string) (value *string, err error) {
	api := session.api

	request := api.GetParameterRequest(&ssm.GetParameterInput{
		Name:           aws.String(name),
		WithDecryption: aws.Bool(true),
	})
	response, err := request.Send(ctx.GetContext())
	if err != nil {
		value = nil
		return
	}
	value = response.GetParameterOutput.Parameter.Value
	return
}
