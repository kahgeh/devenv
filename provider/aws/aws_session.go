package aws

import (
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
)

type Template string

const (
	TemplateParamStoreKeyPath Template = "/allEnvs/%s/pstoreKey"
)

type Session struct {
	config aws.Config
}

type Config struct {
	VpcStackName          string
	EcsSpotFleetStackName string
	EcsClusterStackName   string
	PublicIPStackName     string
	EcsClusterName        string
	EcsSpotFleetPurpose   string
	KeyPairName           string
	HostedZoneName        string
}

var awsComputeConfig = Config{
	VpcStackName:          "DevTest",
	EcsSpotFleetStackName: "GeneralPurposeEcs",
	EcsClusterStackName:   "DevTestEcsCluster",
	PublicIPStackName:     "DevTestPublicIp",
	EcsClusterName:        "DevTest",
	EcsSpotFleetPurpose:   "gp",
}

func CreateAwsSession() (*Session, error) {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		panic("unable to load SDK config, " + err.Error())
	}
	session := &Session{config: cfg}
	awsComputeConfig.KeyPairName = session.GetKeyPairName()
	return session, nil
}

func (session *Session) GetKeyPairName() string {
	return fmt.Sprintf("ecs-instance-%v", *&session.config.Region)
}

func (session *Session) GetComputeConfig() Config {
	return awsComputeConfig
}
