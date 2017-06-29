package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
)

//*******************
//EC2 functions
//*******************

//Delete orphaned hosts
func deleteOrphans(asgName string, awsaz string) bool {
	sess, err := session.NewSession()
	if err != nil {
		panic(err)
	}
	var svcparams = &aws.Config{Region: aws.String(awsaz)}

	if ARN != "" {
		creds := stscreds.NewCredentials(sess, ARN)
		svcparams = &aws.Config{
			Region:      aws.String(awsaz),
			Credentials: creds,
		}
	}
	svc := ec2.New(sess, svcparams)
	if err != nil {
		panic(err)
	}
	resp, err := svc.DescribeInstances(&ec2.DescribeInstancesInput{Filters: []*ec2.Filter{&ec2.Filter{Name: aws.String("tag:orphaned"), Values: []*string{aws.String(asgName)}}}})
	if err != nil {
		panic(err)
	}
	if len(resp.Reservations) == 0 {
		logrus.Info("No orphans")
		return true
	} else if len(resp.Reservations) > 0 {
		for _, reservation := range resp.Reservations {
			for _, instance := range reservation.Instances {
				if *resp.Reservations[0].Instances[0].State.Name == "running" {
					terminateInstance(*instance.InstanceId, awsaz)
					logrus.Info("Deleted orphaned instance: " + *instance.InstanceId)
				}
			}
		}
	}
	return true
}

//Return the name of the most recently added host in an ASG
func newHostName(asgName string, awsaz string) string {
	sess, err := session.NewSession()
	if err != nil {
		panic(err)
	}
	var svcparams = &aws.Config{Region: aws.String(awsaz)}

	if ARN != "" {
		creds := stscreds.NewCredentials(sess, ARN)
		svcparams = &aws.Config{
			Region:      aws.String(awsaz),
			Credentials: creds,
		}
	}
	svc := ec2.New(sess, svcparams)
	if err != nil {
		panic(err)
	}
	resp, err := svc.DescribeInstances(&ec2.DescribeInstancesInput{Filters: []*ec2.Filter{&ec2.Filter{Name: aws.String("tag:aws:autoscaling:groupName"), Values: []*string{aws.String(asgName)}}}})
	if err != nil {
		panic(err)
	}
	var newServerName string
	latestTime, _ := time.Parse(time.RFC822, "01 Jan 00 10:00 UTC")
	if len(resp.Reservations) == 0 {
		logrus.Info("No Servers in ASG")
		return ""
	} else if len(resp.Reservations) > 0 {
		for _, reservation := range resp.Reservations {
			for _, instance := range reservation.Instances {
				logrus.Info("Checking: " + *instance.InstanceId + " State: " + *instance.State.Name)
				if *resp.Reservations[0].Instances[0].State.Name == "running" {
					if latestTime.Before(*instance.LaunchTime) {
						latestTime = *instance.LaunchTime
						newServerName = *instance.PrivateDnsName
						logrus.Info("new recent instance: " + *instance.InstanceId)
					}
				}
			}
		}
	}
	return newServerName
}

//Return the private DNS of the most recently added host in an ASG
func newHostIP(asgName string, awsaz string) string {
	sess, err := session.NewSession()
	if err != nil {
		panic(err)
	}
	var svcparams = &aws.Config{Region: aws.String(awsaz)}

	if ARN != "" {
		creds := stscreds.NewCredentials(sess, ARN)
		svcparams = &aws.Config{
			Region:      aws.String(awsaz),
			Credentials: creds,
		}
	}
	svc := ec2.New(sess, svcparams)
	if err != nil {
		panic(err)
	}
	resp, err := svc.DescribeInstances(&ec2.DescribeInstancesInput{Filters: []*ec2.Filter{&ec2.Filter{Name: aws.String("tag:aws:autoscaling:groupName"), Values: []*string{aws.String(asgName)}}}})
	if err != nil {
		panic(err)
	}
	var newServerIP string
	latestTime, _ := time.Parse(time.RFC822, "01 Jan 00 10:00 UTC")
	if len(resp.Reservations) == 0 {
		logrus.Info("No Servers in ASG")
		return ""
	} else if len(resp.Reservations) > 0 {
		for _, reservation := range resp.Reservations {
			for _, instance := range reservation.Instances {
				logrus.Info("Checking: " + *instance.InstanceId + " State: " + *instance.State.Name)
				if *resp.Reservations[0].Instances[0].State.Name == "running" {
					if latestTime.Before(*instance.LaunchTime) {
						latestTime = *instance.LaunchTime
						newServerIP = *instance.PrivateIpAddress
						logrus.Info("new recent instance: " + *instance.InstanceId)
					}
				}
			}
		}
	}
	return newServerIP
}

//Function to check validity of aws region
func validRegion(awsaz string) bool {
	/*	sess, err := session.NewSession()
			if err != nil {
				panic(err)
			}
		var svcparams = &aws.Config{Region: aws.String(awsaz)}

		if ARN != "" {
			creds := stscreds.NewCredentials(sess, ARN)
			svcparams = &aws.Config{
				Region:      aws.String(awsaz),
				Credentials: creds,
			}
		}
		svc := ec2.New(sess, svcparams)
			if err != nil {
				panic(err)
			}
			regions, err := svc.DescribeRegions(&ec2.DescribeRegionsInput{})

			if err != nil {
				logrus.Info("Invalid Region specified:", awsaz)
			}
			for _, region := range regions.Regions {
				//Check that a valid region has been passed
				if *region.RegionName == awsaz {
					return true
				}
			}*/
	return true
}

//Function to return the name of the ASG associated with the server
func hostASG(h string, awsaz string) string {
	sess, err := session.NewSession()
	if err != nil {
		panic(err)
	}
	var svcparams = &aws.Config{Region: aws.String(awsaz)}

	if ARN != "" {
		creds := stscreds.NewCredentials(sess, ARN)
		svcparams = &aws.Config{
			Region:      aws.String(awsaz),
			Credentials: creds,
		}
	}
	svc := ec2.New(sess, svcparams)
	if err != nil {
		panic(err)
	}

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	resp, err := svc.DescribeInstances(&ec2.DescribeInstancesInput{Filters: []*ec2.Filter{&ec2.Filter{Name: aws.String("private-dns-name"), Values: []*string{aws.String(h)}}}})
	if err != nil {
		panic(err)
	}
	if len(resp.Reservations) == 0 {
		return ""
	} else if len(resp.Reservations) == 1 {
		for _, i := range resp.Reservations[0].Instances {
			var nt string
			for _, t := range i.Tags {
				if *t.Key == "aws:autoscaling:groupName" {
					nt = *t.Value
					return nt
				}
			}
		}
	}
	return ""
}

//Function to return the private-dns-name of the hostASG
func instanceHostname(instanceID string, awsaz string) string {
	sess, err := session.NewSession()
	if err != nil {
		panic(err)
	}
	var svcparams = &aws.Config{Region: aws.String(awsaz)}

	if ARN != "" {
		creds := stscreds.NewCredentials(sess, ARN)
		svcparams = &aws.Config{
			Region:      aws.String(awsaz),
			Credentials: creds,
		}
	}
	svc := ec2.New(sess, svcparams)
	if err != nil {
		panic(err)
	}
	resp, err := svc.DescribeInstances(&ec2.DescribeInstancesInput{Filters: []*ec2.Filter{&ec2.Filter{Name: aws.String("instance-id"), Values: []*string{aws.String(instanceID)}}}})
	if err != nil {
		panic(err)
	}
	if len(resp.Reservations) == 0 {
		return ""
	} else if len(resp.Reservations) == 1 {
		for _, i := range resp.Reservations[0].Instances {
			return *i.PrivateDnsName
		}
	}
	return ""
}

//Function to terminate an instance
func terminateInstance(instanceID string, awsaz string) bool {
	sess, err := session.NewSession()
	if err != nil {
		panic(err)
	}
	var svcparams = &aws.Config{Region: aws.String(awsaz)}

	if ARN != "" {
		creds := stscreds.NewCredentials(sess, ARN)
		svcparams = &aws.Config{
			Region:      aws.String(awsaz),
			Credentials: creds,
		}
	}
	svc := ec2.New(sess, svcparams)
	if err != nil {
		panic(err)
	}
	resp, err := svc.TerminateInstances(&ec2.TerminateInstancesInput{InstanceIds: []*string{aws.String(instanceID)}})
	if err != nil {
		fmt.Println(resp)
		panic(err)
	}
	return true
}

//Function to return the AMI ID that is associated with an instance
func instanceAmiID(i string, awsaz string) string {

	sess, err := session.NewSession()
	if err != nil {
		panic(err)
	}
	var svcparams = &aws.Config{Region: aws.String(awsaz)}

	if ARN != "" {
		creds := stscreds.NewCredentials(sess, ARN)
		svcparams = &aws.Config{
			Region:      aws.String(awsaz),
			Credentials: creds,
		}
	}
	svc := ec2.New(sess, svcparams)
	if err != nil {
		panic(err)
	}

	params := &ec2.DescribeInstancesInput{
		InstanceIds: []*string{
			aws.String(i),
		},
	}
	resp, err := svc.DescribeInstances(params)

	if err != nil {
		fmt.Println(err.Error())
	}
	return *resp.Reservations[0].Instances[0].ImageId
}

//Function to add a tag to an EC2 InstanceIds
func tagInstance(instanceID string, tag string, awsaz string) bool {
	sess, err := session.NewSession()
	if err != nil {
		panic(err)
	}
	var svcparams = &aws.Config{Region: aws.String(awsaz)}

	if ARN != "" {
		creds := stscreds.NewCredentials(sess, ARN)
		svcparams = &aws.Config{
			Region:      aws.String(awsaz),
			Credentials: creds,
		}
	}
	svc := ec2.New(sess, svcparams)
	if err != nil {
		panic(err)
	}

	// Add tags to the instance
	createResult, err := svc.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{aws.String(instanceID)},
		Tags: []*ec2.Tag{
			&ec2.Tag{
				Key:   aws.String("orphaned"),
				Value: aws.String(tag),
			},
		},
	})
	if err != nil {
		log.Println("Could not create tags for instance", instanceID, err)
		fmt.Println(createResult)
		return false
	}

	return true
}

//**************************
//Autoscaling functions
//**************************

//Function to detach an instance from and ASG and add a tag to it
func detachAndTag(instanceID string, asgName string, awsaz string) bool {

	sess, err := session.NewSession()
	if err != nil {
		panic(err)
	}
	var svcparams = &aws.Config{Region: aws.String(awsaz)}

	if ARN != "" {
		creds := stscreds.NewCredentials(sess, ARN)
		svcparams = &aws.Config{
			Region:      aws.String(awsaz),
			Credentials: creds,
		}
	}
	svc := autoscaling.New(sess, svcparams)
	if err != nil {
		panic(err)
	}
	params := &autoscaling.DetachInstancesInput{
		AutoScalingGroupName:           aws.String(asgName), // Required
		ShouldDecrementDesiredCapacity: aws.Bool(true),      // Required
		InstanceIds: []*string{
			aws.String(instanceID),
		},
	}
	resp, err := svc.DetachInstances(params)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(resp)
		fmt.Println(err.Error())
		return false
	}

	tagInstance(instanceID, asgName, awsaz)

	// Pretty-print the response data.
	return true
}

//Function to update the min scale of and ASG
func updateASGMinScale(n string, i int64, awsaz string) bool {

	//reduce the ASG scale by 1
	sess := session.Must(session.NewSession())

	var svcparams = &aws.Config{Region: aws.String(awsaz)}

	if ARN != "" {
		creds := stscreds.NewCredentials(sess, ARN)
		svcparams = &aws.Config{
			Region:      aws.String(awsaz),
			Credentials: creds,
		}
	}
	svc := autoscaling.New(sess, svcparams)

	paramsNewSize := &autoscaling.UpdateAutoScalingGroupInput{
		AutoScalingGroupName: aws.String(n), // Required
		MinSize:              aws.Int64(i),
	}
	asgResp, err := svc.UpdateAutoScalingGroup(paramsNewSize)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(asgResp)
		fmt.Println(err.Error())
		return false
	}

	// Pretty-print the response data.
	return true
}

//LCAmiID Function to return the ami-id of a LaunchConfigurationName
func LCAmiID(l string, awsaz string) string {
	logrus.Info("Launch Config Name: ", l)
	sess := session.Must(session.NewSession())
	var svcparams = &aws.Config{Region: aws.String(awsaz)}

	if ARN != "" {
		creds := stscreds.NewCredentials(sess, ARN)
		svcparams = &aws.Config{
			Region:      aws.String(awsaz),
			Credentials: creds,
		}
	}
	svc := autoscaling.New(sess, svcparams)

	params := &autoscaling.DescribeLaunchConfigurationsInput{
		LaunchConfigurationNames: []*string{
			aws.String(l),
		},
		MaxRecords: aws.Int64(1),
	}
	resp, err := svc.DescribeLaunchConfigurations(params)

	if err != nil {
		// Message from an error.
		fmt.Println(err.Error())
	}
	return *resp.LaunchConfigurations[0].ImageId
}

//Function to return the number of instances in an ASG
func ASGHostCount(asgName string, awsaz string) int {
	sess := session.Must(session.NewSession())
	var svcparams = &aws.Config{Region: aws.String(awsaz)}

	if ARN != "" {
		creds := stscreds.NewCredentials(sess, ARN)
		svcparams = &aws.Config{
			Region:      aws.String(awsaz),
			Credentials: creds,
		}
	}
	svc := autoscaling.New(sess, svcparams)

	params := &autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []*string{
			aws.String(asgName), // Required
			// More values...
		},
		MaxRecords: aws.Int64(1),
	}
	resp, err := svc.DescribeAutoScalingGroups(params)
	if err != nil {
		// Message from an error.
		fmt.Println(err.Error())
	}
	if len(resp.AutoScalingGroups) == 1 {
		return len(resp.AutoScalingGroups[0].Instances)
	}

	return 0
}
