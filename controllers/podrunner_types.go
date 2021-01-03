package controllers

import (
	"github.com/evryfs/github-actions-runner-operator/api/v1alpha1"
	"github.com/google/go-github/v33/github"
	"github.com/redhat-cop/operator-utils/pkg/util"
	"github.com/thoas/go-funk"
	corev1 "k8s.io/api/core/v1"
	"sort"
)

type podRunnerPair struct {
	pod    corev1.Pod
	runner github.Runner
}

type podRunnerPairList struct {
	pairs   []podRunnerPair
	podList corev1.PodList
	runners []*github.Runner
}

func from(podList *corev1.PodList, runners []*github.Runner) podRunnerPairList {
	runnersByName := make(map[string]github.Runner)
	for _, runner := range runners {
		runnersByName[runner.GetName()] = *runner
	}

	podRunnerPairs := podRunnerPairList{
		podList: *podList,
		runners: runners,
	}

	for _, pod := range podList.Items {
		pair := podRunnerPair{
			pod:    pod,
			runner: runnersByName[pod.Name],
		}
		podRunnerPairs.pairs = append(podRunnerPairs.pairs, pair)
	}

	return podRunnerPairs
}

func (r podRunnerPairList) getBusyRunners() []*github.Runner {
	return funk.Filter(r.runners, func(runner *github.Runner) bool {
		return runner.GetBusy()
	}).([]*github.Runner)
}

func (r podRunnerPairList) numBusy() int {
	return len(r.getBusyRunners())
}

func (r podRunnerPairList) allBusy() bool {
	return r.numBusy() == r.numRunners()
}

func (r podRunnerPairList) numPods() int {
	return len(r.podList.Items)
}

func (r podRunnerPairList) numRunners() int {
	return len(r.runners)
}

func (r podRunnerPairList) inSync() bool {
	return r.numPods() == r.numRunners()
}

func (r podRunnerPairList) getIdlePods(sortOrder v1alpha1.SortOrder) []corev1.Pod {
	idles := funk.Filter(r.pairs, func(pair podRunnerPair) bool {
		return !(pair.runner.GetBusy() || util.IsBeingDeleted(&pair.pod))
	}).([]podRunnerPair)
	pods := funk.Map(idles, func(pair podRunnerPair) corev1.Pod {
		return pair.pod
	}).([]corev1.Pod)

	sort.SliceStable(pods, func(i, j int) bool {
		if sortOrder == v1alpha1.LeastRecent {
			return pods[i].CreationTimestamp.Unix() < pods[j].CreationTimestamp.Unix()
		}
		return pods[i].CreationTimestamp.Unix() > pods[j].CreationTimestamp.Unix()
	})

	return pods
}

func (r podRunnerPairList) getPodsBeingDeleted() []podRunnerPair {
	return funk.Filter(r.pairs, func(pair podRunnerPair) bool {
		return util.IsBeingDeleted(&pair.pod)
	}).([]podRunnerPair)
}
