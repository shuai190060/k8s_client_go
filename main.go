package main

import (
	"context"
	"fmt"

	"os"

	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		return map[string]string{err.Error(): deployment.Name}, nil
		// return nil, fmt.Errorf("deployment error:%s", err)
	} else if err != nil && !errors.IsAlreadyExists(err) {
		return nil, fmt.Errorf("deployment error:%s", err)
	}
	return deploymentResponse.Spec.Template.Labels, nil
}
