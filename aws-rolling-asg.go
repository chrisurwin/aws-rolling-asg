package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"io/ioutil"

	"github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/rancher/go-rancher/v2"
)

//Global Variable
var cattleURL = os.Getenv("CATTLE_URL")
var cattleAccessKey = os.Getenv("CATTLE_ACCESS_KEY")
var cattleSecretKey = os.Getenv("CATTLE_SECRET_KEY")
var opts = &client.ClientOpts{
	Url:       cattleURL,
	AccessKey: cattleAccessKey,
	SecretKey: cattleSecretKey,
}
var haASGName = os.Getenv("HA_ASG_NAME")
var haASGRegion = os.Getenv("HA_ASG_REGION")
var projectID = ""
var rancherEnv = ""

//ARN will read in if there is an AWS ARN set
var ARN = os.Getenv("AWS_ARN")

func main() {
	if len(cattleURL) == 0 {
		logrus.Fatalf("CATTLE_URL is not set")
	}

	if len(cattleAccessKey) == 0 {
		logrus.Fatalf("CATTLE_ACCESS_KEY is not set")
	}

	if len(cattleSecretKey) == 0 {
		logrus.Fatalf("CATTLE_SECRET_KEY is not set")
	}
	//Get the Environment
	if os.Getenv("ENVIRONMENT") == "" {
		resp, err := http.Get("http://rancher-metadata/latest/self/stack/environment_name")
		if err != nil {
			fmt.Println("Rancher Metadata not available")
		} else {
			defer resp.Body.Close()
			respOutput, _ := ioutil.ReadAll(resp.Body)
			rancherEnv = string(respOutput)
			fmt.Println("Rancher environment set to: " + rancherEnv)
		}
	} else {
		rancherEnv = os.Getenv("ENVIRONMENT")
	}
	if rancherEnv == "" {
		logrus.Fatal("Rancher Environment not found")
	}

	logrus.Info("Starting Rancher AWS Host cleanup")
	projectID = getProjectID(rancherEnv)
	go startHealthcheck()
	for {
		var returnCode = 0
		returnCode = newFunc("ENV")
		if returnCode == 1 {
			time.Sleep(60 * time.Second)
		} else {
			time.Sleep(5 * time.Minute)
		}
		if haASGName != "" && haASGRegion != "" {
			returnCode = newFunc("HA")
			time.Sleep(60 * time.Second)
			if returnCode == 1 {
				time.Sleep(60 * time.Second)
			} else {
				time.Sleep(5 * time.Minute)
			}
		}
	}
}

//Function to process ASG's in an environment or HA cluster
func newFunc(mode string) int {
	logrus.Info("Checking environment for host image upgrades")

	//Get a list of ASGs
	ASGList := map[string]string{}
	if mode == "ENV" {
		if envHealth(rancherEnv) == "healthy" {
			logrus.Info("Environment " + rancherEnv + " is healthy, continuing")

			ASGList = generateASGList()
		}
		for asg, reg := range ASGList {
			fmt.Println(asg + " " + reg)
		}
	}
	if mode == "HA" {
		ASGList[haASGName] = haASGRegion
	}

	if len(ASGList) > 0 {
		for asg, reg := range ASGList {
			deleteOrphans(asg, reg)
			var launchConfigID = ""
			var amiID = ""

			if asg != "" {
				sess, err := session.NewSession()
				if err != nil {
					panic(err)
				}
				var svcparams = &aws.Config{Region: aws.String(reg)}

				if ARN != "" {
					creds := stscreds.NewCredentials(sess, ARN)

					svcparams = &aws.Config{
						Region:      aws.String(reg),
						Credentials: creds,
					}
				}
				svc := autoscaling.New(sess, svcparams)

				params := &autoscaling.DescribeAutoScalingGroupsInput{
					AutoScalingGroupNames: []*string{
						aws.String(asg),
					},
					MaxRecords: aws.Int64(1),
				}
				resp, err := svc.DescribeAutoScalingGroups(params)

				if err != nil {
					// Print the error, cast err to awserr.Error to get the Code and
					// Message from an error.
					fmt.Println(err.Error())
					return 3
				}
				//Get the AMI id of the Launch Configuration
				if len(resp.AutoScalingGroups) == 0 {
					logrus.Error("Can't find ASG: " + asg)
					return 1
				}
				launchConfigID = *resp.AutoScalingGroups[0].LaunchConfigurationName
				amiID = LCAmiID(launchConfigID, reg)
				logrus.Info("AMI ID of the ASG: ", amiID)

				logrus.Info("Number of Instances in ASG: " + strconv.Itoa(ASGHostCount(asg, reg)))
				//Check each host to see if it is running the correct image
				for _, i := range resp.AutoScalingGroups[0].Instances {
					instanceAMI := instanceAmiID(*i.InstanceId, reg)
					logrus.Info("Instance AMI:", instanceAMI)
					if instanceAMI == amiID {
						logrus.Info("AMI current for host:", i.InstanceId)
					} else {
						logrus.Info("AMI needs updating for host", i.InstanceId)
						// Get the name of the host so it can be reconciled with the rancher instance

						//reduce the ASG scale by 1
						logrus.Info("Removing instance from ASG")
						asgSize := resp.AutoScalingGroups[0].MinSize //get the old size
						asgNewSize := *asgSize - int64(1)
						updateASGMinScale(asg, asgNewSize, reg) //Scale the ASG min down by 1

						detachAndTag(*i.InstanceId, asg, reg) //detach instance from ASG and add a Tag

						updateASGMinScale(asg, *asgSize, reg) //Scale the ASG back up
						time.Sleep(30 * time.Second)
						//Allow for ASG to start a new instance
						for int(*asgSize) > ASGHostCount(asg, reg) {
							time.Sleep(5 * time.Second)
						}
						time.Sleep(30 * time.Second)

						if mode == "ENV" {
							var rancherHostName = instanceHostname(*i.InstanceId, reg)
							logrus.Info("Currently processing: " + rancherHostName)

							var newHostPrivateDNS = newHostName(asg, reg)
							logrus.Info("Repacement Host: " + newHostPrivateDNS)

							logrus.Info("Waiting for the new instance to appear in Rancher")
							for !hostInRancher(newHostPrivateDNS) {
								time.Sleep(5 * time.Second)
							}
							logrus.Info("Host in Rancher, waiting for 1 minute to allow system services to start")
							time.Sleep(60 * time.Second)

							for envHealth(rancherEnv) != "healthy" {
								time.Sleep(5 * time.Second)
							}
							//Evacuate the orphaned host
							logrus.Info("Evacuating the host")
							evacuateHost(rancherHostName)

							time.Sleep(60 * time.Second)
						}
						if mode == "HA" {
							var newHostPrivateIP = newHostIP(asg, reg)
							logrus.Info("Repacement Host IP: " + newHostPrivateIP)

							for !checkInHA(newHostPrivateIP) {
								time.Sleep(10 * time.Second)
							}
						}
						//Delete the AWS host
						logrus.Info("Removing instance")
						terminateInstance(*i.InstanceId, reg)
						if mode == "ENV" {
							//when all services are back to green terminate the instance
							logrus.Info("Waiting for environment to be healthy")
							for envHealth(rancherEnv) != "healthy" {
								time.Sleep(10 * time.Second)
							}
						}

						logrus.Info("Successfully upgraded an instance to the correct image")
						//then next
						return 2
					}
				}
			}
		}
		return 1
	} else {
		logrus.Info("AutoScaling group list empty")
	}
	return 1
}
