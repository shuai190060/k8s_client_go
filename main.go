package main

import (
	"context"
	"fmt"
	"time"

	"os"

	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	var (
		client          *kubernetes.Clientset
		err             error
		deploymentlabel map[string]string
	)
	ctx := context.Background()
	if client, err = GetClient(); err != nil {
		fmt.Printf("Error with k8s client:%s", err)
		os.Exit(1)
	}
	// fmt.Printf("test:%+v", client) // for debug

	if deploymentlabel, err = deploy(ctx, client); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err = waitForPod(ctx, client, deploymentlabel); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Printf("deploy successd, deploy with label: %+v\n", deploymentlabel)

}

func GetClient() (*kubernetes.Clientset, error) {
	// kubeconfig file, i put here with kops generating it
	const kubeconfig = "./kubeconfig"

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	return clientset, err

}

func deploy(ctx context.Context, client *kubernetes.Clientset) (map[string]string, error) {
	var deployment *v1.Deployment
	appFile, err := os.ReadFile("test_deployment.yml")
	if err != nil {
		return nil, fmt.Errorf("err of readfile:%s", err)
	}
	// decode the yml file
	obj, groupVersionKind, err := scheme.Codecs.UniversalDeserializer().Decode(appFile, nil, nil)

	switch obj.(type) {
	case *v1.Deployment:
		//if the type is deployment
		deployment = obj.(*v1.Deployment)
	default:
		return nil, fmt.Errorf("unreconginized type: %s", groupVersionKind)
	}

	_, err = client.AppsV1().Deployments("default").Get(ctx, deployment.Name, metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		deploymentResponse, err := client.AppsV1().Deployments("default").Create(ctx, deployment, metav1.CreateOptions{})
		if err != nil {
			return nil, fmt.Errorf("deployment get error:%s", err)
		}
		return deploymentResponse.Spec.Template.Labels, nil
	} else if err != nil && !errors.IsNotFound(err) {
		return nil, fmt.Errorf("deployment get error:%s", err)
	}

	// decode := schema.Codecs.UniversalDeserializer().Decode
	deploymentResponse, err := client.AppsV1().Deployments("default").Create(ctx, deployment, metav1.CreateOptions{})
	if err != nil && errors.IsAlreadyExists(err) {
		existingDeployment, getErr := client.AppsV1().Deployments("default").Update(ctx, deployment, metav1.UpdateOptions{})
		if getErr != nil {
			return nil, fmt.Errorf("error fetch deployment: %s", getErr)
		}
		return existingDeployment.Spec.Template.Labels, nil
		// return nil, fmt.Errorf("deployment error:%s", err)
	} else if err != nil && !errors.IsAlreadyExists(err) {
		return nil, fmt.Errorf("deployment error:%s", err)
	}
	return deploymentResponse.Spec.Template.Labels, nil
}

func waitForPod(ctx context.Context, client *kubernetes.Clientset, deploymentslabels map[string]string) error {
	for {

		validatedLabels, err := labels.ValidatedSelectorFromSet(deploymentslabels)
		if err != nil {
			return fmt.Errorf("ValidatedSelectorFromSet error:%s", err)
		}
		podList, err := client.CoreV1().Pods("default").List(ctx, metav1.ListOptions{
			LabelSelector: validatedLabels.String(),
		})
		if err != nil {
			return fmt.Errorf("pod list error: %s", err)
		}

		podsRunning := 0
		for _, pod := range podList.Items {
			if pod.Status.Phase == "Running" {
				podsRunning++
			}
		}
		fmt.Printf("waiting for pods to become ready.(running %d /%d)", podsRunning, len(podList.Items))

		if podsRunning > 0 && podsRunning == len(podList.Items) {
			break
		}
		time.Sleep(5 * time.Second)

	}
	return nil
}
