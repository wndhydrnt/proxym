package elb

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	awsElb "github.com/aws/aws-sdk-go/service/elb"
	"github.com/kelseyhightower/envconfig"
	"github.com/wndhydrnt/proxym/log"
	"github.com/wndhydrnt/proxym/manager"
	"github.com/wndhydrnt/proxym/types"
	"strings"
)

type Config struct {
	Enabled bool
	Regions string
}

type ElbSynchroniser struct {
	ec2Clients map[string]*ec2.EC2
	elbClients map[string]*awsElb.ELB
}

func (e *ElbSynchroniser) Generate(services []*types.Service) error {
	for _, service := range services {
		elbName, ok := service.Attributes["proxym:elb:name"]
		if ok == false {
			continue
		}

		elbRegion, ok := service.Attributes["proxym:elb:region"]
		if ok == false {
			continue
		}

		ec2Client, err := e.ec2ClientByRegion(elbRegion)
		if err != nil {
			log.ErrorLog.Error("%s", err)
			continue
		}

		elbClient, err := e.elbClientByRegion(elbRegion)
		if err != nil {
			log.ErrorLog.Error("%s", err)
			continue
		}

		ec2Instances, err := e.ec2Instances(ec2Client, service.Hosts)
		if err != nil {
			log.ErrorLog.Error(fmt.Sprintf("Error finding EC2 instances of service %s: %s", service.Id, err))
			continue
		}

		elbInstances, err := e.currentElbInstances(elbClient, elbName)
		if err != nil {
			log.ErrorLog.Error("Error finding ELB instances of service %s: %s", service.Id, err)
			continue
		}

		instancesToRegister, instancesToDeregister := e.compareState(ec2Instances, elbInstances)

		if len(instancesToRegister) > 0 {
			err = e.registerInstances(elbClient, instancesToRegister, elbName)
			if err != nil {
				log.ErrorLog.Error("Error registering instances to ELB: %s", err)
				continue
			}
		}

		if len(instancesToDeregister) > 0 {
			err = e.deregisterInstances(elbClient, instancesToDeregister, elbName)
			if err != nil {
				log.ErrorLog.Error("Error deregistering instances to ELB: %s", err)
			}
		}
	}

	return nil
}

func (e *ElbSynchroniser) ec2Instances(client *ec2.EC2, hosts []types.Host) ([]*ec2.Instance, error) {
	privateIpAddresses := []*string{}

	for _, host := range hosts {
		privateIpAddresses = append(privateIpAddresses, &host.Ip)
	}

	filterName := "private-ip-address"

	out, err := client.DescribeInstances(
		&ec2.DescribeInstancesInput{
			Filters: []*ec2.Filter{&ec2.Filter{Name: &filterName, Values: privateIpAddresses}},
		},
	)
	if err != nil {
		return nil, err
	}

	return out.Reservations[0].Instances, nil
}

func (e *ElbSynchroniser) ec2ClientByRegion(region string) (*ec2.EC2, error) {
	c, ok := e.ec2Clients[region]

	if ok == false {
		return nil, errors.New(fmt.Sprintf("No EC2 client registered for region '%s'", region))
	}

	return c, nil
}

func (e *ElbSynchroniser) elbClientByRegion(region string) (*awsElb.ELB, error) {
	c, ok := e.elbClients[region]

	if ok == false {
		return nil, errors.New(fmt.Sprintf("No ELB client registered for region '%s'", region))
	}

	return c, nil
}

func (e *ElbSynchroniser) compareState(ec2Instances []*ec2.Instance, elbInstances []*awsElb.Instance) ([]*awsElb.Instance, []*awsElb.Instance) {
	elbInstanceIds := make(map[*string]*awsElb.Instance)
	ec2InstanceIds := make(map[*string]*ec2.Instance)
	toRegister := []*awsElb.Instance{}
	toDeregister := []*awsElb.Instance{}

	for _, ec2Instance := range ec2Instances {
		ec2InstanceIds[ec2Instance.InstanceID] = ec2Instance
	}

	for _, elbInstance := range elbInstances {
		elbInstanceIds[elbInstance.InstanceID] = elbInstance
	}

	for _, ec2Instance := range ec2Instances {
		_, ok := elbInstanceIds[ec2Instance.InstanceID]
		if ok == false {
			toRegister = append(toRegister, &awsElb.Instance{InstanceID: ec2Instance.InstanceID})
		}
	}

	for _, elbInstance := range elbInstances {
		_, ok := ec2InstanceIds[elbInstance.InstanceID]
		if ok == false {
			toDeregister = append(toDeregister, elbInstance)
		}
	}

	return toRegister, toDeregister
}

func (e *ElbSynchroniser) currentElbInstances(client *awsElb.ELB, name string) ([]*awsElb.Instance, error) {
	out, err := client.DescribeLoadBalancers(&awsElb.DescribeLoadBalancersInput{LoadBalancerNames: []*string{&name}})
	if err != nil {
		return nil, err
	}

	if len(out.LoadBalancerDescriptions) > 1 {
		return nil, errors.New(fmt.Sprintf("ELB name '%s' is ambiguous - received %d results", name, len(out.LoadBalancerDescriptions)))
	}

	return out.LoadBalancerDescriptions[0].Instances, nil
}

func (e *ElbSynchroniser) deregisterInstances(elbClient *awsElb.ELB, instances []*awsElb.Instance, name string) error {
	_, err := elbClient.DeregisterInstancesFromLoadBalancer(
		&awsElb.DeregisterInstancesFromLoadBalancerInput{
			Instances:        instances,
			LoadBalancerName: &name,
		},
	)

	return err
}

func (e *ElbSynchroniser) registerInstances(elbClient *awsElb.ELB, instances []*awsElb.Instance, name string) error {
	_, err := elbClient.RegisterInstancesWithLoadBalancer(
		&awsElb.RegisterInstancesWithLoadBalancerInput{
			Instances:        instances,
			LoadBalancerName: &name,
		},
	)

	return err
}

func NewElbSynchroniser(c *Config) *ElbSynchroniser {
	ec2Clients := make(map[string]*ec2.EC2)
	elbClients := make(map[string]*awsElb.ELB)

	regions := strings.Split(c.Regions, ",")

	for _, region := range regions {
		ec2Clients[region] = ec2.New(&aws.Config{Region: region})
		elbClients[region] = awsElb.New(&aws.Config{Region: region})
	}

	return &ElbSynchroniser{
		ec2Clients: ec2Clients,
		elbClients: elbClients,
	}
}

func init() {
	var c Config

	envconfig.Process("proxym_proxy", &c)

	if c.Enabled {
		e := NewElbSynchroniser(&c)

		manager.AddConfigGenerator(e)
	}
}
