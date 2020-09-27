package aws

import (
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/awserr"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/kahgeh/devenv/logger"
	"github.com/kahgeh/devenv/utils/ctx"
	"os"
)

func (session *Session) deleteKeyPair() {
	log := logger.NewTaskLogger()
	defer log.LogDone()

	keyPairName := session.GetKeyPairName()
	api := ec2.New(session.config)
	request := api.DeleteKeyPairRequest(&ec2.DeleteKeyPairInput{
		KeyName: aws.String(keyPairName),
	})
	_, err := request.Send(ctx.GetContext())
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == "InvalidKeyPair.NotFound" {
				log.Infof("Key pair %q does not exists", keyPairName)
				return
			}
			log.Debug(aerr.Message())
			log.Fail(aerr.Code())
			return
		}
		log.Failf("%v", err)
		return
	}

	sshFolderPath := getSshFolderPath()
	sshPrivateKeyFilePath := fmt.Sprintf("%s/%s.pem", sshFolderPath, keyPairName)
	log.Infof("deleting ssh key %s...", sshPrivateKeyFilePath)
	if stat, _ := os.Stat(sshPrivateKeyFilePath); stat != nil {
		err := os.Remove(sshPrivateKeyFilePath)
		if err != nil {
			log.Failf("fail to delete %s", sshPrivateKeyFilePath)
		}
	}
	log.Succeed()
}

func (session *Session) deleteVpc() {
	log := logger.NewTaskLogger()
	defer log.LogDone()
	config := session.GetComputeConfig()
	stackName := config.VpcStackName
	stack := NewStack(stackName, session)
	stack.Delete()
	log.Succeed()
}

func (session *Session) deleteEcsCluster() {
	log := logger.NewTaskLogger()
	defer log.LogDone()
	config := session.GetComputeConfig()
	stackName := config.EcsClusterStackName

	stack := NewStack(stackName, session)
	api := stack.api
	stack.Delete()
	err := api.WaitUntilStackDeleteComplete(ctx.GetContext(), &cloudformation.DescribeStacksInput{
		StackName: &stack.name,
	})

	if err != nil {
		log.Debug(err.Error())
		log.Fail("waiting ecs cluster deletion failed")
		return
	}

	log.Succeed()
}

func (session *Session) deleteSpotFleet() {
	log := logger.NewTaskLogger()
	defer log.LogDone()
	config := session.GetComputeConfig()
	stackName := config.EcsSpotFleetStackName
	stack := NewStack(stackName, session)
	api := stack.api
	stack.Delete()
	err := api.WaitUntilStackDeleteComplete(ctx.GetContext(), &cloudformation.DescribeStacksInput{
		StackName: &stack.name,
	})
	if err != nil {
		log.Debug(err.Error())
		log.Fail("waiting spot fleet deletion failed")
		return
	}
	log.Succeed()
}

// Delete tears down the compute
func (session *Session) Delete() {
	session.deleteSpotFleet()
	session.deleteEcsCluster()
	session.deleteVpc()
	session.deleteKeyPair()
}
