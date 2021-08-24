package main

import (
	"encoding/base64"
	"fmt"

	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
	dcs "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dcs/v2"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dcs/v2/model"
	region "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dcs/v2/region"
)

// find DCS instance under user's account by name
// return DCS host, isNoPasswordAccess, decoded password and error
func FindDCS(req *DCSConnectRequest) (string, string, string, error) {
	realAK, err := base64.StdEncoding.DecodeString(req.AK)
	if err != nil {
		return "", "", "", err
	}
	realSK, err := base64.StdEncoding.DecodeString(req.SK)
	if err != nil {
		return "", "", "", err
	}
	realPassword, err := base64.StdEncoding.DecodeString(req.Credential)
	if err != nil {
		return "", "", "", err
	}

	auth := basic.NewCredentialsBuilder().
		WithAk(string(realAK)).
		WithSk(string(realSK)).
		Build()

	client := dcs.NewDcsClient(
		dcs.DcsClientBuilder().
			WithRegion(region.ValueOf("cn-north-4")).
			WithCredential(auth).
			Build())

	request := &model.ListInstancesRequest{}
	response, err := client.ListInstances(request)
	if err != nil {
		fmt.Println(err)
	}
	if *response.InstanceNum == 0 {
		return "", "", "", fmt.Errorf("your account does not have any DCS instances")
	}

	if req.DCSName == "" {
		dcsList := *response.Instances
		v := dcsList[0]
		return fmt.Sprintf("%v:%v", &v.Ip, &v.Port), *v.NoPasswordAccess, string(realPassword), nil
	}

	for _, v := range *response.Instances {
		if *v.Name == req.DCSName {
			if *v.Status == "RUNNING" {
				// value := v.Ip + ":"+ &v.Port
				return fmt.Sprintf("%v:%v", *v.Ip, *v.Port), *v.NoPasswordAccess, string(realPassword), nil
			}
			return "", "", "", fmt.Errorf("%s is not running", req.DCSName)

		}
	}

	return "", "", "", fmt.Errorf("%s not found", req.DCSName)
}
