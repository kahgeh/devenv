package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/kahgeh/devenv/logger"
	"github.com/kahgeh/devenv/utils/ctx"
)

func (session *Session) removeEcsInstance() {
	log := logger.NewTaskLogger()
	defer log.LogDone()
	config := session.GetComputeConfig()
	stackName := config.EcsSpotFleetStackName

	spotFleetRequestID := session.getStackOutputValue(
		fmt.Sprintf("%s-spotfleetrequest", stackName),
		stackName)

	api := ec2.New(session.config)
	modificationRequest := api.ModifySpotFleetRequestRequest(&ec2.ModifySpotFleetRequestInput{
		SpotFleetRequestId: spotFleetRequestID,
		TargetCapacity:     aws.Int64(0),
	})

	_, err := modificationRequest.Send(ctx.GetContext())
	if err != nil {
		log.Fail("failed to remove instance")
		return
	}
	log.Succeed()

}

func (session *Session) deletePublicIP() {
	log := logger.NewTaskLogger()
	defer log.LogDone()
	config := session.GetComputeConfig()
	stackName := config.PublicIPStackName
	stack := NewStack(stackName, session)
	api := stack.api
	stack.Delete()
	err := api.WaitUntilStackDeleteComplete(ctx.GetContext(), &cloudformation.DescribeStacksInput{
		StackName: &stack.name,
	})
	if err != nil {
		log.Debug(err.Error())
		log.Fail("fail to wait for public ip deletion")
		return
	}
	log.Succeed()
}

// Stop terminates the ecs instance and remove any attached resources
func (session *Session) Stop() {
	session.removeEcsInstance()
	session.deletePublicIP()
}
