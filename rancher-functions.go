package main

import (
	"github.com/Sirupsen/logrus"
	"github.com/rancher/go-rancher/v2"
)

//Function to return the health state of an environemnt
func envHealth(e string) string {

	c, err := client.NewRancherClient(opts)
	if err != nil {
		logrus.Error("Error with client connection")
	} else {
		stacks, err := c.Project.List(nil)
		if err != nil {
			logrus.Error("Error with stack list")
		} else {

			for _, p := range stacks.Data {
				if p.Name == rancherEnv {
					return p.HealthState
				}
			}

			logrus.Error("Environment " + e + " not found")
			return "NotFound"
		}
	}
	logrus.Error("Client Connection Error")
	return "Client Connection Error"
}

//Function to check if an IP address is one associated with HA cluster
func checkInHA(hostIP string) bool {

	c, err := client.NewRancherClient(opts)
	if err != nil {
		logrus.Error("Error with client connection")
	}

	nodes, err := c.ClusterMembership.List(nil)
	if err != nil {
		logrus.Error("Error getting HA cluster membership")
	}

	for _, h := range nodes.Data {
		if hostIP == findIP(h.Config) {
			return true
		}
	}
	return false
}

//Function to return the projectID of an environemnt
func getProjectID(e string) string {

	c, err := client.NewRancherClient(opts)
	if err != nil {
		logrus.Error("Error with client connection")
	}
	stacks, err := c.Project.List(nil)
	if err != nil {
		logrus.Error("Error with stack list")
	}

	for _, p := range stacks.Data {
		if p.Name == rancherEnv {
			logrus.Info("Environment projectid: " + p.Id)
			return p.Id
		}
	}

	logrus.Error("Environment " + e + " not found")
	return "NotFound"
}

//Function to return the number of hosts in an environemnt
func hostCount() int {
	c, err := client.NewRancherClient(opts)
	if err != nil {
		logrus.Error("Error with client connection")
	}

	//Get a list of Hosts

	hosts, err := c.Host.List(nil)
	var i = 0
	for _, h := range hosts.Data {
		if h.AccountId == projectID { //we are only interested in hosts in this environment
			i++
		}
	}
	return i
}

//Function to return if host is in environment
func hostInRancher(hostName string) bool {
	c, err := client.NewRancherClient(opts)
	if err != nil {
		logrus.Error("Error with client connection")
	}

	//Get a list of Hosts

	hosts, err := c.Host.List(nil)
	for _, h := range hosts.Data {
		if h.Hostname == hostName { //we are only interested in hosts in this environment
			return true
		}
	}
	return false
}

//Function to evacuate a host
func evacuateHost(hostName string) bool {
	c, err := client.NewRancherClient(opts)
	if err != nil {
		logrus.Error("Error with client connection")
		return false
	}
	//Get a list of Hosts
	hosts, err := c.Host.List(nil)
	for _, h := range hosts.Data {
		if h.Hostname == hostName {
			c.Host.ActionEvacuate(&h)
		}
	}
	return true
}
