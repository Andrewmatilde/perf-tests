/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package util

import (
	"fmt"

	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/pkg/util/system"
)

// GetSchedulableUntainedNodesNumber returns number of nodes in the cluster.
func GetSchedulableUntainedNodesNumber(c clientset.Interface) (int, error) {
	nodeList, err := c.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return 0, err
	}
	numNodes := 0
	for i := range nodeList.Items {
		if isNodeSchedulable(&nodeList.Items[i]) && isNodeUntainted(&nodeList.Items[i]) {
			numNodes++
		}
	}
	return numNodes, err
}

// Node is schedulable if:
// 1) doesn't have "unschedulable" field set
// 2) it's Ready condition is set to true
// 3) doesn't have NetworkUnavailable condition set to true
func isNodeSchedulable(node *corev1.Node) bool {
	nodeReady := isNodeConditionSetAsExpected(node, corev1.NodeReady, true, false)
	networkReady := isNodeConditionUnset(node, corev1.NodeNetworkUnavailable) ||
		isNodeConditionSetAsExpected(node, corev1.NodeNetworkUnavailable, false, true)
	return !node.Spec.Unschedulable && nodeReady && networkReady
}

// Tests whether node doesn't have any taint with "NoSchedule" or "NoExecute" effect.
func isNodeUntainted(node *corev1.Node) bool {
	for i := range node.Spec.Taints {
		if node.Spec.Taints[i].Effect == corev1.TaintEffectNoSchedule || node.Spec.Taints[i].Effect == corev1.TaintEffectNoExecute {
			return false
		}
	}
	return true
}

func isNodeConditionSetAsExpected(node *corev1.Node, conditionType corev1.NodeConditionType, wantTrue, silent bool) bool {
	// Check the node readiness condition (logging all).
	for _, cond := range node.Status.Conditions {
		// Ensure that the condition type and the status matches as desired.
		if cond.Type == conditionType {
			if wantTrue == (cond.Status == corev1.ConditionTrue) {
				return true
			}
			if !silent {
				glog.Infof("Condition %s of node %s is %v instead of %t. Reason: %v, message: %v",
					conditionType, node.Name, cond.Status == corev1.ConditionTrue, wantTrue, cond.Reason, cond.Message)
			}
			return false
		}
	}
	if !silent {
		glog.Infof("Couldn't find condition %v on node %v", conditionType, node.Name)
	}
	return false
}

func isNodeConditionUnset(node *corev1.Node, conditionType corev1.NodeConditionType) bool {
	for _, cond := range node.Status.Conditions {
		if cond.Type == conditionType {
			return false
		}
	}
	return true
}

// GetMasterName returns master node name.
func GetMasterName(c clientset.Interface) (string, error) {
	nodeList, err := c.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return "", err
	}
	for _, node := range nodeList.Items {
		if system.IsMasterNode(node.Name) {
			return node.Name, nil
		}
	}
	return "", fmt.Errorf("master node not found")
}

// GetMasterExternalIP returns master node external ip.
func GetMasterExternalIP(c clientset.Interface) (string, error) {
	nodeList, err := c.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return "", err
	}
	for _, node := range nodeList.Items {
		if system.IsMasterNode(node.Name) {
			for _, address := range node.Status.Addresses {
				if address.Type == corev1.NodeExternalIP {
					return address.Address, nil
				}
			}
			return "", fmt.Errorf("extrnal IP of the master not found")
		}
	}
	return "", fmt.Errorf("master node not found")
}
