package controllers

import (
	"fmt"
	"sort"
	"time"

	"github.com/evryfs/github-actions-runner-operator/api/v1alpha1"
	"github.com/google/go-github/v59/github"
	"github.com/redhat-cop/operator-utils/pkg/util"
	"github.com/samber/lo"
	corev1 "k8s.io/api/core/v1"
)

type podRunnerPair struct {
	pod    corev1.Pod
	runner github.Runner
}

func (r *podRunnerPair) getNamespacedName() string {
	return fmt.Sprintf("%s/%s", r.pod.Namespace, r.pod.Name)
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
	return lo.Filter(r.runners, func(runner *github.Runner, _ int) bool {
		return runner.GetBusy()
	})
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

func (r podRunnerPairList) numIdle() int {
	return r.numRunners() - r.numBusy()
}

func (r podRunnerPairList) getIdles(sortOrder v1alpha1.SortOrder, minTTL time.Duration) []podRunnerPair {
	idles := lo.Filter(r.pairs, func(pair podRunnerPair, _ int) bool {
		return !(pair.runner.GetBusy() || util.IsBeingDeleted(&pair.pod)) && time.Now().After(pair.pod.CreationTimestamp.Add(minTTL))
	})

	sort.SliceStable(idles, func(i, j int) bool {
		if sortOrder == v1alpha1.LeastRecent {
			return idles[i].pod.CreationTimestamp.Unix() < idles[j].pod.CreationTimestamp.Unix()
		}
		return idles[i].pod.CreationTimestamp.Unix() > idles[j].pod.CreationTimestamp.Unix()
	})

	return idles
}

func (r podRunnerPairList) getPodsBeingDeletedOrEvictedOrCompleted() []podRunnerPair {
	return lo.Filter(r.pairs, func(pair podRunnerPair, _ int) bool {
		return util.IsBeingDeleted(&pair.pod) || isEvicted(&pair.pod) || isCompleted(&pair.pod)
	})
}
