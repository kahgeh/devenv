package aws

import (
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/kahgeh/devenv/logger"
	"github.com/kahgeh/devenv/utils/ctx"
)

var ChangeSetCompletionStatuses = []cloudformation.ChangeSetStatus{
	cloudformation.ChangeSetStatusDeleteComplete,
	cloudformation.ChangeSetStatusCreateComplete,
	cloudformation.ChangeSetStatusFailed,
}

type ChangeSet struct {
	name    string
	id      string
	stack   Stack
	details *cloudformation.DescribeChangeSetResponse
}

func containStatus(status cloudformation.ChangeSetStatus, statuses []cloudformation.ChangeSetStatus) bool {
	for _, statusItem := range statuses {
		if statusItem == status {
			return true
		}
	}
	return false

}

func waitForChangesetCompletion(poll func() (bool, *cloudformation.DescribeChangeSetResponse), result chan *cloudformation.DescribeChangeSetResponse) {
	ticker := time.NewTicker(2 * time.Second)
	defer func() { ticker.Stop() }()
	for {
		select {
		case <-ctx.GetContext().Done():
			result <- nil
			return
		case <-ticker.C:
			if end, details := poll(); end {
				result <- details
				return
			}
		}
	}
}

func (changeset *ChangeSet) WaitTillExecutable() *ChangeSet {
	log := logger.New()
	defer log.LogDone()
	if changeset == nil {
		return nil
	}
	// todo - switch to waiter
	resultChannel := make(chan *cloudformation.DescribeChangeSetResponse)
	go waitForChangesetCompletion(func() (bool, *cloudformation.DescribeChangeSetResponse) {
		description := changeset.Describe()
		return containStatus(description.Status, ChangeSetCompletionStatuses), description
	}, resultChannel)
	result := <-resultChannel
	if result == nil {
		return nil
	}

	if result.Status == cloudformation.ChangeSetStatusFailed {
		log.Failf("Fail to create changeset %s", changeset.name)
		return nil
	}

	return &ChangeSet{
		stack:   changeset.stack,
		id:      changeset.id,
		details: result,
	}
}

func (changeset *ChangeSet) Describe() *cloudformation.DescribeChangeSetResponse {
	stackName := changeset.stack.name
	api := changeset.stack.api
	request := api.DescribeChangeSetRequest(&cloudformation.DescribeChangeSetInput{
		StackName:     &stackName,
		ChangeSetName: aws.String(changeset.name),
	})
	response, err := request.Send(ctx.GetContext())
	if err != nil {
		return nil
	}
	return response
}

func (changeset *ChangeSet) String() string {
	if changeset == nil {
		return ""
	}
	stackName := changeset.stack.name
	return fmt.Sprintf("change set id=%v name=%v of stack %v", changeset.id, changeset.name, stackName)
}

func getChangesSummary(changes []cloudformation.Change) string {
	var sb strings.Builder
	for _, change := range changes {
		sb.WriteString(fmt.Sprintf("%s %s\n", string(change.ResourceChange.Action),
			*change.ResourceChange.LogicalResourceId))
	}
	return sb.String()
}

func (changeset *ChangeSet) SendExecuteRequest() *string {
	log := logger.New()
	defer log.LogDone()
	if changeset == nil {
		return nil
	}
	stackName := changeset.stack.name
	api := changeset.stack.api
	token := fmt.Sprintf("%s-%v", stackName, time.Now().UnixNano())
	changeSummary := getChangesSummary(changeset.details.Changes)
	log.Debugf("Changes :\n%s", changeSummary)

	request := api.ExecuteChangeSetRequest(&cloudformation.ExecuteChangeSetInput{
		ChangeSetName:      &changeset.id,
		ClientRequestToken: &token,
	})

	_, err := request.Send(ctx.GetContext())
	if err != nil {
		log.Infof("error executeChange- %s", err)
		return nil
	}

	return &token
}
