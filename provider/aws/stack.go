package aws

import (
	"fmt"
	"github.com/kahgeh/devenv/provider/aws/errors"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/awserr"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/kahgeh/devenv/logger"
	"github.com/kahgeh/devenv/utils/ctx"
)

type StackUpdateCompletion string

type Stack struct {
	api     *cloudformation.Client
	name    string
	details *cloudformation.Stack
}

func NewStack(name string, awsSession *Session) *Stack {
	return &Stack{
		api:  cloudformation.New(awsSession.config),
		name: name,
	}
}

// CreateChangeSet create a changeset
func (stack *Stack) CreateChangeSet(name string, changesetType cloudformation.ChangeSetType, templateBody string, parameters []cloudformation.Parameter) *ChangeSet {
	log := logger.New()
	if stack == nil {
		return nil
	}
	stackName := stack.name
	token := fmt.Sprintf("%s-%v", stackName, time.Now().UnixNano())

	request := stack.api.CreateChangeSetRequest(&cloudformation.CreateChangeSetInput{
		StackName:     aws.String(stackName),
		Capabilities:  []cloudformation.Capability{"CAPABILITY_IAM", "CAPABILITY_NAMED_IAM", "CAPABILITY_AUTO_EXPAND"},
		TemplateBody:  &templateBody,
		ChangeSetType: changesetType,
		ChangeSetName: aws.String(name),
		ClientToken:   aws.String(token),
		Parameters:    parameters,
	})

	response, err := request.Send(ctx.GetContext())
	if err != nil {
		log.Debug(err.Error())
		log.Fail("error creating change set request")
		return nil
	}
	description, err := stack.Describe()
	if err != nil {
		log.Fail("unable to get description of stack '%s'", stackName)
		return nil
	}
	log.Infof("created changeset, name=%q", name)

	return &ChangeSet{
		id:   *response.Id,
		name: name,
		stack: Stack{
			api:     stack.api,
			name:    stackName,
			details: description,
		},
	}
}

// Delete remove stack, waits for completion
func (stack *Stack) Delete() *string {
	log := logger.New()
	if stack == nil {
		return nil
	}
	stackName := stack.name
	token := fmt.Sprintf("%s-%v", stackName, time.Now().UnixNano())
	api := stack.api
	request := api.DeleteStackRequest(&cloudformation.DeleteStackInput{
		StackName:          aws.String(stackName),
		ClientRequestToken: aws.String(token),
	})
	_, err := request.Send(ctx.GetContext())
	if err != nil {
		log.Info(err)
		if awsErr, ok := err.(awserr.Error); ok {
			log.Info(awsErr.Code())
			log.Debug(awsErr.Error())
			log.Debug(awsErr.Message())
		}

		return &token
	}
	err = api.WaitUntilStackDeleteComplete(ctx.GetContext(),
		&cloudformation.DescribeStacksInput{
			StackName: aws.String(stackName),
		})
	if err != nil {
		log.Debug(err.Error())
		log.Fail("fail to wait for stack deletion to complete")
		return nil
	}
	return &token
}

func (stack *Stack) Describe() (description *cloudformation.Stack, err error) {
	request := stack.api.DescribeStacksRequest(&cloudformation.DescribeStacksInput{
		StackName: aws.String(stack.name),
	})
	response, err := request.Send(ctx.GetContext())
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "ValidationError" &&
				strings.Contains(awsErr.Message(), "does not exist") {
				return nil, nil
			}
		} else {
			return nil, err
		}
	}
	if response == nil {
		return nil, &errors.NoResponse{Message: "empty response from describe stack"}
	}

	return &response.Stacks[0], nil
}

// GetEvents retrieves the events for a specific operation
func (stack *Stack) GetEvents(opToken string) (events []cloudformation.StackEvent, err error) {
	request := stack.api.DescribeStackEventsRequest(&cloudformation.DescribeStackEventsInput{
		StackName: aws.String(stack.name),
	})
	response, err := request.Send(ctx.GetContext())
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "ValidationError" &&
				strings.Contains(awsErr.Message(), "does not exist") {
				return nil, nil
			}
		}
		return nil, err
	}
	for _, event := range response.StackEvents {
		if event.ClientRequestToken != nil &&
			*event.ClientRequestToken == opToken {
			events = append(events, event)
		}
	}

	return events, nil
}

func getNameFromStackFileName(actionName string,stackFileName string) string{
	return fmt.Sprintf("%s-%s", actionName, strings.Replace(
		strings.TrimRight(stackFileName, ".yml"),
		"/", "-", -1))
}

// Create creates stack, reports success if successfully create as well as if it already exist
// This function waits for completion
func (stack *Stack) Create(stackFileName string, parameters []cloudformation.Parameter) error {
	log := logger.New()
	defer log.LogDone()
	stackName := stack.name
	changeSetName := getNameFromStackFileName("create", stackFileName)
	templateBody, err := getCfnTemplateContent(stackFileName)
	if err != nil {
		log.Failf("cannot retrieve %s", stackFileName)
	}
	log.Debugf("template body \n%s", templateBody)

	cfnStack, err := stack.Describe()
	if err != nil {
		log.Debugf("unexpected error occured while checking if stack '%s' exist ", stackName)
		return err
	}

	if cfnStack != nil {
		log.Debugf("Stack %v already exists", stackName)
		return nil
	}

	opToken := stack.
		CreateChangeSet(changeSetName, cloudformation.ChangeSetTypeCreate, templateBody, parameters).
		WaitTillExecutable().
		SendExecuteRequest()
	api := stack.api
	err = api.WaitUntilStackCreateComplete(ctx.GetContext(), &cloudformation.DescribeStacksInput{
		StackName: &stack.name,
	})
	if err != nil {
		return err
	}

	events, err := stack.GetEvents(*opToken)
	var eventsSummary strings.Builder
	for _, event := range events {
		eventsSummary.WriteString(fmt.Sprintf("%s [%s] %s\n",
			event.Timestamp.Format(time.Kitchen),
			event.ResourceStatus, *event.LogicalResourceId))
	}
	log.Debugf("events :\n%s", eventsSummary.String())
	return nil
}

// Update creates stack, reports success if successfully create as well as if it already exist
// This function waits for completion
func (stack *Stack) Update(stackFileName string, parameters []cloudformation.Parameter) error {
	log := logger.New()
	defer log.LogDone()
	stackName := stack.name
	changeSetName := getNameFromStackFileName("update", stackFileName)

	templateBody, err := getCfnTemplateContent(stackFileName)
	if err != nil {
		log.Failf("cannot retrieve %s", stackFileName)
	}
	log.Debugf("template body \n%s", templateBody)

	_, err = stack.Describe()
	if err != nil {
		log.Debugf("unexpected error occurred while checking if stack '%s' exist ", stackName)
		return err
	}

	opToken := stack.
		CreateChangeSet(
			changeSetName,
			cloudformation.ChangeSetTypeUpdate,
			templateBody,
			parameters).
		WaitTillExecutable().
		SendExecuteRequest()
	api := stack.api
	err = api.WaitUntilStackUpdateComplete(ctx.GetContext(),
		&cloudformation.DescribeStacksInput{
			StackName: &stack.name,
		})

	if err == nil {
		return err
	}

	events, err := stack.GetEvents(*opToken)
	var eventsSummary strings.Builder
	for _, event := range events {
		eventsSummary.WriteString(fmt.Sprintf("%s [%s] %s\n",
			event.Timestamp.Format(time.Kitchen),
			event.ResourceStatus, *event.LogicalResourceId))
	}
	log.Debugf("events :\n%s", eventsSummary.String())
	return nil
}
