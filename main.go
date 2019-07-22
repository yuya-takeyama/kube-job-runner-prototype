package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
)

type applyResult struct {
	Metadata applyResultMetadata `json:"metadata"`
}

type applyResultMetadata struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type podMetadata struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type containerStatus struct {
	Ready bool                              `json:"ready"`
	State map[string]map[string]interface{} `json:"state"`
}

type podStatus struct {
	ContainerStatuses []containerStatus `json:"containerStatuses"`
}

type pod struct {
	Metadata podMetadata `json:"metadata"`
	Status   podStatus   `json:"status"`
}

type podItems struct {
	Items []pod `json:"items"`
}

func main() {
	file := os.Args[1]
	applyCmd := exec.Command("kubectl", "apply", "-f", file, "-o", "json")
	applyBuf := new(bytes.Buffer)
	applyCmd.Stdout = applyBuf
	applyCmd.Stderr = os.Stderr
	applyErr := applyCmd.Run()
	if applyErr != nil {
		panic(applyErr)
	}

	log.Println("kubectl apply result:")
	fmt.Println(applyBuf.String())
	log.Println("---")

	var ar applyResult
	jdErr := json.Unmarshal(applyBuf.Bytes(), &ar)
	if jdErr != nil {
		panic(jdErr)
	}

	jobName := ar.Metadata.Name
	jobNamespace := ar.Metadata.Namespace

	var podName string

	for {
		items, getJobPodsErr := getJobPods(jobNamespace, jobName)
		if getJobPodsErr != nil {
			panic(getJobPodsErr)
		}

		if items.Items[0].Status.ContainerStatuses[0].State["waiting"] == nil {
			podName = items.Items[0].Metadata.Name
			break
		}
	}

	logCmd := exec.Command("kubectl", "logs", "-n", jobNamespace, podName, "-f", "--timestamps")
	logCmd.Stdout = os.Stdout
	logCmd.Stderr = os.Stderr
	logErr := logCmd.Run()
	if logErr != nil {
		panic(logErr)
	}

	po, poErr := getPod(jobNamespace, podName)
	if poErr != nil {
		panic(poErr)
	}

	exitCode := int(po.Status.ContainerStatuses[0].State["terminated"]["exitCode"].(float64))
	os.Exit(exitCode)
}

func getJobPods(namespace string, jobName string) (*podItems, error) {
	getPodCmd := exec.Command("kubectl", "get", "pods", "-n", namespace, "--selector=job-name="+jobName, "-o", "json")
	getPodBuf := new(bytes.Buffer)
	getPodCmd.Stdout = getPodBuf
	getPodErr := getPodCmd.Run()
	if getPodErr != nil {
		return nil, getPodErr
	}

	log.Println("kubectl get pods result:")
	fmt.Println(getPodBuf.String())
	log.Println("---")

	var pjr podItems
	pjErr := json.Unmarshal(getPodBuf.Bytes(), &pjr)
	if pjErr != nil {
		return nil, pjErr
	}

	return &pjr, nil
}

func getPod(namespace string, podName string) (*pod, error) {
	getPodCmd := exec.Command("kubectl", "get", "pods", "-n", namespace, podName, "-o", "json")
	getPodBuf := new(bytes.Buffer)
	getPodCmd.Stdout = getPodBuf
	getPodErr := getPodCmd.Run()
	if getPodErr != nil {
		return nil, getPodErr
	}
	var po pod
	pjErr := json.Unmarshal(getPodBuf.Bytes(), &po)
	if pjErr != nil {
		return nil, pjErr
	}

	return &po, nil
}
