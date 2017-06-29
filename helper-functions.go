package main

import (
	"regexp"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/go-rancher/v2"
)

func findIP(input string) string {
	numBlock := "(25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])"
	regexPattern := numBlock + "\\." + numBlock + "\\." + numBlock + "\\." + numBlock

	regEx := regexp.MustCompile(regexPattern)
	return regEx.FindString(input)
}

func generateASGList() map[string]string {

	logrus.Info("Generating ASG..")
	ASGList := map[string]string{}
	c, err := client.NewRancherClient(opts)
	if err != nil {
		logrus.Error("Error with client connection")
	}

	hosts, err := c.Host.List(nil)
	if err != nil {
		logrus.Error("Error with host list")
	}

	var awsaz = ""
	//Get the name of ASG TODO:Will have to update this to cater for more than one ASG in an environment
	for _, h := range hosts.Data {
		if h.State == "active" && h.AccountId == projectID { //we are only interested in running hosts in this environment
			if h.Driver == "" || h.Driver == "null" {
				awsaz = strings.Split(h.Hostname, ".")[1]
			} else if h.Driver == "amazonec2" {
				awsaz = h.Amazonec2Config.Region
			}
			logrus.Info("Hostname" + h.Hostname + " ASG:" + hostASG(h.Hostname, awsaz))

			if validRegion(awsaz) {
				ASGList[hostASG(h.Hostname, awsaz)] = awsaz
			}
		}
	}
	return ASGList
}
