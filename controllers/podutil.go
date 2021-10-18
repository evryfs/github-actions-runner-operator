package controllers

import (
	v1 "k8s.io/api/core/v1"
	"strings"
)

func isEvicted(pod *v1.Pod) bool {
	return strings.Contains(pod.Status.Reason, "Evicted")
}

func isCompleted(pod *v1.Pod) bool {
	return pod.Status.Phase == v1.PodSucceeded
}
