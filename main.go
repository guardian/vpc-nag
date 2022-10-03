package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
)

type PrismSubnet struct {
	AvailabilityZone        string `json:"availabilityZone"`
	AvailableIPAddressCount int64  `json:"availableIpAddressCount"`
	CapacityIPAddressCount  int64  `json:"capacityIpAddressCount"`
	CidrBlock               string `json:"cidrBlock"`
	OwnerID                 string `json:"ownerId"`
	State                   string `json:"state"`
	SubnetArn               string `json:"subnetArn"`
	SubnetID                string `json:"subnetId"`
	IsPublic                bool   `json:"isPublic"`
}

type PrismVPC struct {
	VPCID     string            `json:"vpcId"`
	AccountID string            `json:"accountId"`
	State     string            `json:"state"`
	IsDefault bool              `json:"default"`
	Subnets   []PrismSubnet     `json:"subnets"`
	Tags      map[string]string `json:"tags"`
	Meta      struct {
		Origin struct {
			Region string `json:"region"`
		} `json:"origin"`
	} `json:"meta"`
}

type PrismResponse struct {
	Data struct {
		VPCs []PrismVPC `json:"vpcs"`
	} `json:"data"`
}

func main() {
	accountID := flag.String("accountID", "", "Specify account (ID) to audit.")
	flag.Parse()

	if *accountID == "" {
		fmt.Println("Missing required argument: accountID")
	}

	resp, err := http.Get("https://prism.gutools.co.uk/vpcs")
	check(err, "GET from PRISM failed")

	data, err := io.ReadAll(resp.Body)
	check(err, "unable to read prism response body")
	defer resp.Body.Close()

	prismResponse := PrismResponse{}
	err = json.Unmarshal(data, &prismResponse)
	check(err, "unable to unmarshal")

	accountVPCs := Filter(prismResponse.Data.VPCs, func(vpc PrismVPC) bool {
		return vpc.AccountID == *accountID
	})

	for _, vpc := range accountVPCs {
		if vpc.AccountID != *accountID {
			continue
		}

		complianceErrs := checkCompliance(vpc)
		if len(complianceErrs) > 0 {
			reportCompliance(vpc, complianceErrs)
		}
	}

	nonEuWest1 := Filter(accountVPCs, func(vpc PrismVPC) bool {
		return vpc.Meta.Origin.Region != "eu-west-1"
	})

	if len(nonEuWest1) > 0 {
		fmt.Printf("The following VPCs were ignored as are in non-standard regions:\n")
		for _, vpc := range nonEuWest1 {
			fmt.Printf("\t%s (%s)\n", vpc.VPCID, vpc.Meta.Origin.Region)
		}
	}

}

func checkCompliance(vpc PrismVPC) []error {
	errs := []error{}

	if vpc.Meta.Origin.Region != "eu-west-1" {
		return errs // ignore
	}

	if vpc.IsDefault {
		errs = append(errs, errors.New("is Default VPC"))
		return errs // don't bother checking other errors
	}

	// has 3 public subnets and 3 private subnets
	publicSubnets := Filter(vpc.Subnets, func(subnet PrismSubnet) bool {
		return subnet.IsPublic
	})

	privateSubnets := Filter(vpc.Subnets, func(subnet PrismSubnet) bool {
		return !subnet.IsPublic
	})

	if len(publicSubnets) != 3 {
		errs = append(errs, fmt.Errorf("expected 3 public subnets, found %d", len(publicSubnets)))
	}

	if len(privateSubnets) != 3 {
		errs = append(errs, fmt.Errorf("expected 3 private subnets, found %d", len(privateSubnets)))
	}

	return errs
}

func Filter[A any](items []A, pred func(A) bool) []A {
	var out []A

	for _, item := range items {
		if pred(item) {
			out = append(out, item)
		}
	}

	return out
}

func reportCompliance(vpc PrismVPC, errors []error) {
	fmt.Printf("Failed: %s (%s)\n", vpc.VPCID, vpc.Meta.Origin.Region)

	for _, err := range errors {
		fmt.Printf("\t%s\n", err)
	}

	fmt.Println()
}

func check(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %v", msg, err)
	}
}
