package aws

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/wish/kops-controller/fallbackidentity"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog"
)

// fallbackIdentifier identifies a node from EC2
type fallbackIdentifier struct {
	// client is the ec2 interface
	ec2Client ec2iface.EC2API
}

func New() (fallbackidentity.Identifier, error) {
	config := aws.NewConfig()
	config = config.WithCredentialsChainVerboseErrors(true)

	s, err := session.NewSession(config)
	if err != nil {
		return nil, fmt.Errorf("error starting new AWS session: %v", err)
	}
	s.Handlers.Send.PushFront(func(r *request.Request) {
		// Log requests
		klog.V(4).Infof("AWS API Request: %s/%s", r.ClientInfo.ServiceName, r.Operation.Name)
	})

	metadata := ec2metadata.New(s, config)

	region, err := metadata.Region()
	if err != nil {
		return nil, fmt.Errorf("error querying ec2 metadata service (for region): %v", err)
	}

	ec2Client := ec2.New(s, config.WithRegion(region))

	return &fallbackIdentifier{
		ec2Client: ec2Client,
	}, nil
}

func (i *fallbackIdentifier) IdentifyNode(ctx context.Context, node *corev1.Node) (map[string]string, error) {
	providerID := node.Spec.ProviderID
	if providerID == "" {
		return nil, fmt.Errorf("providerID was not set for node %s", node.Name)
	}
	if !strings.HasPrefix(providerID, "aws://") {
		return nil, fmt.Errorf("providerID %q not recognized for node %s", providerID, node.Name)
	}

	tokens := strings.Split(strings.TrimPrefix(providerID, "aws://"), "/")
	if len(tokens) != 3 {
		return nil, fmt.Errorf("providerID %q not recognized for node %s", providerID, node.Name)
	}

	//zone := tokens[1]
	instanceID := tokens[2]

	// Based on node-authorizer code
	instance, err := i.getInstance(instanceID)
	if err != nil {
		return nil, err
	}

	labels := make(map[string]string)
	for _, tag := range instance.Tags {
		if strings.HasPrefix(aws.StringValue(tag.Key), "k8s:labels:") {
			labels[aws.StringValue(tag.Key)[len("k8s:labels:"):]] = aws.StringValue(tag.Value)
		}
	}

	return labels, nil
}

// getInstance queries EC2 for the instance with the specified ID, returning an error if not found
func (i *fallbackIdentifier) getInstance(instanceID string) (*ec2.Instance, error) {
	// Based on node-authorizer code
	resp, err := i.ec2Client.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: aws.StringSlice([]string{instanceID}),
	})
	if err != nil {
		return nil, fmt.Errorf("error from ec2 DescribeInstances request: %v", err)
	}

	// @check we found some instances
	if len(resp.Reservations) <= 0 || len(resp.Reservations[0].Instances) <= 0 {
		return nil, fmt.Errorf("missing instance id: %s", instanceID)
	}
	if len(resp.Reservations[0].Instances) > 1 {
		return nil, fmt.Errorf("found multiple instances with instance id: %s", instanceID)
	}

	instance := resp.Reservations[0].Instances[0]
	return instance, nil
}