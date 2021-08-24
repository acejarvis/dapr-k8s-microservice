package main

import (
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (k *KubeClient) ConnectDCS(req *DCSConnectRequest) (string, error) {

	// find DCS
	redisHost, noPasswordAccess, password, err := FindDCS(req)
	if err != nil {
		return "", err
	}
	redisPassword := ""
	if noPasswordAccess == "false" {
		redisPassword = password
	}

	// connect to DCS
	stateRedisConfig := []map[string]interface{}{}
	stateRedisConfig = append(stateRedisConfig, map[string]interface{}{
		"name":  "redisHost",
		"value": redisHost,
	})
	stateRedisConfig = append(stateRedisConfig, map[string]interface{}{
		"name":  "redisPassword",
		"value": redisPassword,
	})
	redis := map[string]interface{}{
		"apiVersion": "dapr.io/v1alpha1",
		"kind":       "Component",
		"metadata": map[string]interface{}{
			"name": "statestore",
		},
		"spec": map[string]interface{}{
			"type":     "state.redis",
			"version":  "v1",
			"metadata": stateRedisConfig,
		},
	}

	yaml, err := ToUnstructured(redis)
	if err != nil {
		return "", err
	}

	meta, err := k.ApplyWithNamespaceOverride(yaml, "default")
	if err != nil {
		return "", err
	}
	b, _ := json.Marshal(meta)

	return string(b), nil
}

func (k *KubeClient) DisconnectDCS() (string, error) {
	err := k.DeleteResourceByKindAndNameAndNamespace("Component", "statestore", "default", metav1.DeleteOptions{})
	if err != nil {
		return "", err
	}
	return "Dapr StateStore Disconnected.", nil
}

func (k *KubeClient) CreateAppDeploy(deployment map[string]interface{}, connect *DCSConnectRequest) (string, error) {

	// connect to DCS
	redisResult, err := k.ConnectDCS(connect)
	if err != nil {
		return "", err
	}
	// parse Deployment template
	deploymentYAML, err := ToUnstructured(deployment)
	if err != nil {
		return "", err
	}
	metadata := deploymentYAML.Object["metadata"].(map[string]interface{})
	app := metadata["labels"].(map[string]interface{})["app"]
	containerPort := deploymentYAML.Object["spec"].(map[string]interface{})["template"].(map[string]interface{})["spec"].(map[string]interface{})["containers"].([]interface{})[0].(map[string]interface{})["ports"].([]interface{})[0].(map[string]interface{})["containerPort"]

	// construct Service template
	portsConfig := []map[string]interface{}{}
	portsConfig = append(portsConfig, map[string]interface{}{
		"protocol":   "TCP",
		"port":       80,
		"targetPort": containerPort,
	})
	service := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Service",
		"metadata":   metadata,
		"spec": map[string]interface{}{
			"selector": map[string]interface{}{
				"app": app,
			},
			"ports": portsConfig,
			"type":  "LoadBalancer",
		},
	}

	serviceYAML, err := ToUnstructured(service)
	if err != nil {
		return "", err
	}

	// apply Service
	serviceResult, err := k.ApplyWithNamespaceOverride(serviceYAML, "default")
	if err != nil {
		return "", err
	}
	serviceJson, _ := json.Marshal(serviceResult)
	// apply Deployment
	deploymentResult, err := k.ApplyWithNamespaceOverride(deploymentYAML, "default")
	if err != nil {
		return "", err
	}
	deploymentJson, _ := json.Marshal(deploymentResult)

	return "App Created \n" + redisResult + "\n" + string(serviceJson) + "\n" + string(deploymentJson), nil
}

func (k *KubeClient) DeleteAppDeploy(namespace string, name string) (string, error) {
	// delete Service
	err := k.DeleteResourceByKindAndNameAndNamespace("Service", name, namespace, metav1.DeleteOptions{})
	if err != nil {
		return "", err
	}

	// delete Deployment
	err = k.DeleteResourceByKindAndNameAndNamespace("Deployment", name, namespace, metav1.DeleteOptions{})
	if err != nil {
		return "", err
	}

	// diconnect DCS
	result, err := k.DisconnectDCS()
	if err != nil {
		return "", err
	}
	str := result + "App has been deleted."
	return fmt.Sprintln(str), nil
}
