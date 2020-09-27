package aws

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/kahgeh/devenv/utils/ctx"
)

func (session *Session) DescribeService(serviceArn string, clusterName string) (*ecs.DescribeServicesResponse, error) {
	svc := ecs.New(session.config)
	input := &ecs.DescribeServicesInput{
		Services: []string{
			serviceArn,
		},
		Cluster: aws.String(clusterName),
	}

	req := svc.DescribeServicesRequest(input)
	result, err := req.Send(ctx.GetContext())
	if err != nil {
		return nil, err
	}
	return result, nil
}
