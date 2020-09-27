package aws

import (
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/kahgeh/devenv/provider/types"
	"io/ioutil"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/awserr"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/kahgeh/devenv/fixed"
	"github.com/kahgeh/devenv/logger"
	"github.com/kahgeh/devenv/utils"
	"github.com/kahgeh/devenv/utils/ctx"
	"github.com/mitchellh/go-homedir"
)

func getSshFolderPath() string {
	home, err := homedir.Dir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return fmt.Sprintf("%v/.ssh", home)
}

func getFrontProxyPath() string {
	configFolderPath := fixed.GetConfigFolderPath()
	return fmt.Sprintf("%s/aws/%s", configFolderPath, "front-proxy")
}

func getCfnTemplateContent(cfnFileName string) (string, error) {
	configFolderPath := fixed.GetConfigFolderPath()
	cfnTemplatePath := fmt.Sprintf("%s/aws/%s", configFolderPath, cfnFileName)
	content, err := ioutil.ReadFile(cfnTemplatePath)
	if err != nil {
		return "", err
	}
	limit := 51200
	if len(content) > limit {
		return "", fmt.Errorf("template %s is over the %v limit", cfnTemplatePath, limit)
	}
	return string(content), nil
}

func (session *Session) createKeyPair() {
	log := logger.NewTaskLogger()
	defer log.LogDone()

	keyPairName := session.GetKeyPairName()
	request := ec2.New(session.config).CreateKeyPairRequest(&ec2.CreateKeyPairInput{
		KeyName: aws.String(keyPairName),
	})
	result, err := request.Send(ctx.GetContext())

	sshFolderPath := getSshFolderPath()
	sshPrivateKeyFilePath := fmt.Sprintf("%s/%s.pem", sshFolderPath, keyPairName)
	if aerr, ok := err.(awserr.Error); ok {
		if aerr.Code() == "InvalidKeyPair.Duplicate" {
			if _, err := os.Stat(sshPrivateKeyFilePath); os.IsNotExist(err) {
				log.Failf("Key pair %q already exists, but '.pem' does not exist", keyPairName, sshPrivateKeyFilePath)
				return
			}
			log.Succeedf("Key pair %q already exists", keyPairName)
			return
		}
		log.Debug(aerr.Message())
		log.Fail(aerr.Code())
		return
	}
	if result == nil {
		log.Fail("no response from creating key pair request")
		return
	}
	keyPair := result
	log.Debugf("Created key pair %q %s\n%s\n",
		*keyPair.KeyName, *keyPair.KeyFingerprint,
		*keyPair.KeyMaterial)

	log.Info("saving private key to ssh folder...")
	err = utils.CreateFolderIfNotExist(sshFolderPath)
	if err != nil {
		log.Failf("fail to ensure %q exist", sshFolderPath)
		return
	}

	err = ioutil.WriteFile(sshPrivateKeyFilePath, []byte(*keyPair.KeyMaterial), 0600)
	if err != nil {
		log.Failf("Failed saving private key to %s", sshPrivateKeyFilePath)
		return
	}
	log.Succeedf("Saved private key to %s", sshPrivateKeyFilePath)
}

func (session *Session) createVpc() {
	log := logger.NewTaskLogger()
	defer log.LogDone()

	config := session.GetComputeConfig()
	stackName := config.VpcStackName

	cfnClient := cloudformation.New(session.config)
	stack := &Stack{name: stackName, api: cfnClient}
	err := stack.Create("vpc.yml", []cloudformation.Parameter{})
	if err != nil {
		log.Fail("fail to create vpc")
	}
	log.Succeed()
}

func (session *Session) createEcsSpotFleet(envName string, domainName string) {
	log := logger.NewTaskLogger()
	defer log.LogDone()

	config := session.GetComputeConfig()
	stackName := config.EcsSpotFleetStackName
	clusterName := config.EcsClusterName
	keyPairName := config.KeyPairName
	purpose := config.EcsSpotFleetPurpose

	cfnClient := cloudformation.New(session.config)
	stack := &Stack{name: stackName, api: cfnClient}
	err := stack.Create("spotFleet.yml", []cloudformation.Parameter{
		{
			ParameterKey:   aws.String("EcsClusterName"),
			ParameterValue: aws.String(clusterName),
		},
		{
			ParameterKey:   aws.String("KeyName"),
			ParameterValue: aws.String(keyPairName),
		},
		{
			ParameterKey:   aws.String("Environment"),
			ParameterValue: aws.String(envName),
		},
		{
			ParameterKey:   aws.String("FleetType"),
			ParameterValue: aws.String(purpose),
		},
		{
			ParameterKey:   aws.String("DomainName"),
			ParameterValue: aws.String(domainName),
		},
	})
	if err != nil {
		log.Fail("fail to create spot fleet")
	}
	log.Succeed()
}

func (session *Session) createEcsCluster(envName string) {
	log := logger.NewTaskLogger()
	defer log.LogDone()

	config := session.GetComputeConfig()
	stackName := config.EcsClusterStackName
	clusterName := config.EcsClusterName

	cfnClient := cloudformation.New(session.config)
	stack := &Stack{name: stackName, api: cfnClient}
	err := stack.Create("ecsCluster.yml", []cloudformation.Parameter{
		{
			ParameterKey:   aws.String("ClusterName"),
			ParameterValue: aws.String(clusterName),
		},
		{
			ParameterKey:   aws.String("Environment"),
			ParameterValue: aws.String(envName),
		},
	})
	if err != nil {
		log.Fail("fail to create ecs cluster")
	}
	log.Succeed()
}

func (session *Session) savePstoreKey(keyName string, path string) {
	log := logger.NewTaskLogger()
	defer log.LogDone()

	apiKms := kms.New(session.config)
	describeKeyRequest := apiKms.DescribeKeyRequest(&kms.DescribeKeyInput{
		KeyId: aws.String(keyName),
	})

	keyDescription, err := describeKeyRequest.Send(ctx.GetContext())
	if err != nil {
		log.Debug(err.Error())
		log.Failf("fail to get %q key arn", keyName)
		return
	}

	keyArn := keyDescription.DescribeKeyOutput.KeyMetadata.Arn

	apiSsm := ssm.New(session.config)
	putParamRequest := apiSsm.PutParameterRequest(&ssm.PutParameterInput{
		Value:     keyArn,
		Name:      aws.String(path),
		Type:      ssm.ParameterTypeString,
		Overwrite: aws.Bool(true),
	})

	putParamResponse, err := putParamRequest.Send(ctx.GetContext())
	if err != nil {
		log.Debug(err.Error())
		log.Fail("fail to save aws/ssm key arn to parameter store")
		return
	}

	log.Debug(putParamResponse.PutParameterOutput.String())
	log.Succeed()
}

func (session *Session) Initialise(parameters *types.InitialisationParameters) {
	log := logger.New()
	defer log.LogDone()
	envName := parameters.EnvironmentName
	domainName := parameters.DomainName
	session.createKeyPair()
	session.createVpc()
	session.createEcsCluster(envName)
	session.createEcsSpotFleet(envName, domainName)
	pstoreKeyPath := fmt.Sprintf(string(TemplateParamStoreKeyPath), envName)
	session.savePstoreKey("alias/aws/ssm", pstoreKeyPath)
}
