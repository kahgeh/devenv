package aws

import (
	"fmt"
	"github.com/kahgeh/devenv/provider/types"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/kahgeh/devenv/logger"
	"github.com/kahgeh/devenv/utils/ctx"
)

type waitResult struct {
	output *string
	err    error
}

func filterOutputs(outputs []cloudformation.Output, predicate func(cloudformation.Output) bool) []cloudformation.Output {
	b := outputs[:0]

	for _, output := range outputs {
		if predicate(output) {
			b = append(b, output)
		}
	}
	return b
}

func (session *Session) createPublicIP(hostedZoneName string, domainName string) *string {
	log := logger.NewTaskLogger()
	defer log.LogDone()
	cfg := session.GetComputeConfig()
	stackName := cfg.PublicIPStackName
	log.Debugf("hostedZoneName=%q", hostedZoneName)
	cfnClient := cloudformation.New(session.config)
	stack := &Stack{name: stackName, api: cfnClient}
	err := stack.Create("publicIp.yml", []cloudformation.Parameter{
		{
			ParameterKey:   aws.String("HostedZoneName"),
			ParameterValue: aws.String(hostedZoneName),
		},
		{
			ParameterKey:   aws.String("DomainName"),
			ParameterValue: aws.String(domainName),
		},
	})
	if err != nil {
		log.Debug(err.Error())
		log.Fail("fail to create public Ip")
	}
	publicIP := session.getStackOutputValue("PublicIp", stackName)
	log.Infof("public ip %q is available and assigned to app.%s", *publicIP, domainName)
	log.Succeed()
	return publicIP
}

func (session *Session) getStackOutputValue(name string, stackName string) *string {
	cfnClient := cloudformation.New(session.config)
	stack := &Stack{name: stackName, api: cfnClient}
	description, _ := stack.Describe()
	result := filterOutputs(description.Outputs, func(output cloudformation.Output) bool {
		return *output.ExportName == name
	})
	return result[0].OutputValue
}

func (session *Session) getStackOutputValueByKey(name string, stackName string) *string {
	cfnClient := cloudformation.New(session.config)
	stack := &Stack{name: stackName, api: cfnClient}
	description, _ := stack.Describe()
	result := filterOutputs(description.Outputs, func(output cloudformation.Output) bool {
		return *output.OutputKey == name
	})
	return result[0].OutputValue
}

func wait(resultChannel chan *waitResult, predicate func() (*string, error)) {
	ticker := time.NewTicker(5 * time.Second)
	allowedRetries := 3
	defer func() {
		ticker.Stop()
		close(resultChannel)
	}()
	for {
		select {
		case <-ctx.GetContext().Done():
			return
		case <-ticker.C:
			result, err := predicate()
			if allowedRetries > 0 && err != nil {
				allowedRetries--
			}
			if allowedRetries == 0 && err != nil {
				resultChannel <- &waitResult{
					err:    err,
					output: nil,
				}
			}
			if result != nil {
				resultChannel <- &waitResult{
					err:    err,
					output: result,
				}
				return
			}
		}
	}
}

func (session *Session) addEcsInstance() *string {
	log := logger.NewTaskLogger()
	defer log.LogDone()

	cfg := session.GetComputeConfig()
	stackName := cfg.EcsSpotFleetStackName

	log.Info("get spot fleet details...")
	spotFleetRequestID := session.getStackOutputValue(
		fmt.Sprintf("%s-spotfleetrequest", stackName),
		stackName)

	api := ec2.New(session.config)
	describeRequest := api.DescribeSpotFleetInstancesRequest(&ec2.DescribeSpotFleetInstancesInput{
		SpotFleetRequestId: spotFleetRequestID,
	})

	descriptionResponse, err := describeRequest.Send(ctx.GetContext())
	if err != nil {
		log.Fail("fail to get spot fleet instance")
		return nil
	}

	if len(descriptionResponse.ActiveInstances) > 0 {
		log.Info("there's already one instance")
		log.Succeed()
		return descriptionResponse.ActiveInstances[0].InstanceId
	}

	log.Info("starting ecs instance...")
	modificationRequest := api.ModifySpotFleetRequestRequest(&ec2.ModifySpotFleetRequestInput{
		SpotFleetRequestId: spotFleetRequestID,
		TargetCapacity:     aws.Int64(1),
	})

	modificationResponse, err := modificationRequest.Send(ctx.GetContext())
	if err != nil {
		log.Fail("fail to allocate new ECS instance")
		return nil
	}

	if !*modificationResponse.Return {
		log.Debug(modificationResponse.String())
		log.Failf("fail to allocate new ECS instance")
		return nil
	}

	log.Info("waiting for ecs instance to be running...")
	c := make(chan *waitResult)
	go wait(c, func() (*string, error) {
		request := api.DescribeSpotFleetInstancesRequest(&ec2.DescribeSpotFleetInstancesInput{
			SpotFleetRequestId: spotFleetRequestID,
		})

		response, err := request.Send(ctx.GetContext())
		if err != nil {
			return nil, err
		}

		if len(response.ActiveInstances) > 0 {
			return response.ActiveInstances[0].InstanceId, nil
		}
		return nil, nil
	})
	result := <-c
	if result != nil && result.err != nil {
		log.Debug(result.err.Error())
		log.Fail("fail to get spot fleet information")
		return nil
	}
	if result == nil {
		log.Fail("get spot fleet returns empty information")
		return nil
	}
	instanceId := *result.output
	err = api.WaitUntilSystemStatusOk(ctx.GetContext(), &ec2.DescribeInstanceStatusInput{
		InstanceIds: []string{instanceId},
	})

	if err != nil {
		log.Failf("waiting for %s to reach OK status failed", instanceId)
		return nil
	}

	log.Succeedf("instanceId %q started", instanceId)
	return result.output
}

func (session *Session) attachPublicIPToEcsInstance(publicIP *string, instanceID *string) {
	log := logger.NewTaskLogger()
	defer log.LogDone()
	api := ec2.New(session.config)

	request := api.AssociateAddressRequest(&ec2.AssociateAddressInput{
		InstanceId: instanceID,
		PublicIp:   publicIP,
	})
	response, err := request.Send(ctx.GetContext())
	if err != nil {
		log.Failf("fail to attach %q to %q", publicIP, instanceID)
		return
	}
	log.Debugf("associationId id=%q", *response.AssociationId)
	log.Succeed()
}

func (session *Session) waitTillInstanceRunning(instanceID string) {
	log := logger.NewTaskLogger()
	defer log.LogDone()
	api := ec2.New(session.config)

	c := make(chan *waitResult)
	go wait(c, func() (*string, error) {
		request := api.DescribeInstanceStatusRequest(&ec2.DescribeInstanceStatusInput{
			InstanceIds: []string{instanceID},
		})
		response, err := request.Send(ctx.GetContext())
		if err != nil {
			return nil, err
		}

		if response.InstanceStatuses[0].InstanceState.Name == ec2.InstanceStateNameRunning {
			return aws.String(string(ec2.InstanceStateNameRunning)), nil
		}

		return nil, nil
	})
	result := <-c
	if result.err != nil {
		log.Debug(result.err.Error())
		log.Fail("unexpected error while waiting for instance to be running")
		return
	}
	log.Succeed()
}

// Start starts up the required compute and public interface
func (session *Session) Start(parameters *types.StartParameters) {
	hostedZoneName := parameters.HostedZoneName
	domainName := parameters.DomainName
	publicIP := session.createPublicIP(hostedZoneName, domainName)
	instanceID := session.addEcsInstance()
	session.waitTillInstanceRunning(*instanceID)
	session.attachPublicIPToEcsInstance(publicIP, instanceID)
}
