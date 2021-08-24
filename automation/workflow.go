package main

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (k *KubeClient) ConnectRedis(redisHost string, redisPassword string) (string, error) {
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
		return "ERROR", err
	}

	meta, err := k.ApplyWithNamespaceOverride(yaml, "default")
	if err != nil {
		return "ERROR", err
	}
	fmt.Println(meta)
	return fmt.Sprintln(meta), nil
}

func (k *KubeClient) DisconnectRedis(namespace string, name string) (string, error) {
	err := k.DeleteResourceByKindAndNameAndNamespace("Component", "statestore", "default", metav1.DeleteOptions{})
	if err != nil {
		return "ERROR Disconnecting Redis", err
	}
	return "Disconnected Redis", nil
}

func (k *KubeClient) CreateAppDeploy(deployment map[string]interface{}) (string, error) {

	redisHost := "172.16.0.197:6379"
	redisPassword := "Cloud@123"

	meta, err := k.ConnectRedis(redisHost, redisPassword)
	fmt.Println(meta)

	// deployment, _ := readJSON()
	deploymentYAML, err := ToUnstructured(deployment)
	if err != nil {
		return "ERROR", err
	}
	metadata := deploymentYAML.Object["metadata"].(map[string]interface{})
	app := metadata["labels"].(map[string]interface{})["app"]
	containerPort := deploymentYAML.Object["spec"].(map[string]interface{})["template"].(map[string]interface{})["spec"].(map[string]interface{})["containers"].([]interface{})[0].(map[string]interface{})["ports"].([]interface{})[0].(map[string]interface{})["containerPort"]

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
		return "ERROR", err
	}

	result, err := k.ApplyWithNamespaceOverride(serviceYAML, "default")
	if err != nil {
		return "ERROR", err
	}
	fmt.Println(result)
	result1, err := k.ApplyWithNamespaceOverride(deploymentYAML, "default")
	if err != nil {
		return "ERROR", err
	}
	fmt.Println(result1)

	return fmt.Sprintln(result, result1), nil
}

func (k *KubeClient) DeleteAppDeploy(namespace string, name string) (string, error) {
	err := k.DeleteResourceByKindAndNameAndNamespace("Service", name, namespace, metav1.DeleteOptions{})
	if err != nil {
		return "ERROR", err
	}

	err = k.DeleteResourceByKindAndNameAndNamespace("Deployment", name, namespace, metav1.DeleteOptions{})
	if err != nil {
		return "ERROR", err
	}

	result, err := k.DisconnectRedis("default", "statestore")
	if err != nil {
		return "ERROR", err
	}
	str := result + "App has been deleted."
	return fmt.Sprintln(str), nil
}
