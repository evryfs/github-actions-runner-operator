package controllers

import (
	"github.com/evryfs/github-actions-runner-operator/api/v1alpha1"
	"github.com/google/go-github/v40/github"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"testing"
	"time"
)

var podList = v1.PodList{
	TypeMeta: metav1.TypeMeta{},
	ListMeta: metav1.ListMeta{},
	Items: []v1.Pod{
		{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:              "name1",
				CreationTimestamp: metav1.NewTime(time.Now().Add(-time.Minute)),
			},
			Spec:   v1.PodSpec{},
			Status: v1.PodStatus{},
		},
		{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:              "name2",
				CreationTimestamp: metav1.NewTime(time.Now()),
			},
			Spec:   v1.PodSpec{},
			Status: v1.PodStatus{},
		},
	},
}

var runners = []*github.Runner{
	{
		ID:     nil,
		Name:   pointer.StringPtr("name1"),
		OS:     nil,
		Status: nil,
		Busy:   nil,
		Labels: nil,
	},
	{
		ID:     nil,
		Name:   pointer.StringPtr("name2"),
		OS:     nil,
		Status: nil,
		Busy:   nil,
		Labels: nil,
	},
}

func TestPodRunnerPairList(t *testing.T) {
	testCases := []struct {
		podList v1.PodList
		runners []*github.Runner
		inSync  bool
	}{
		{podList, runners, true},
		{v1.PodList{Items: podList.Items[:1]}, runners, false},
		{podList, runners[:1], false},
	}

	for _, tc := range testCases {
		podRunnerPairList := from(&tc.podList, tc.runners)
		assert.Equal(t, podRunnerPairList.inSync(), tc.inSync)
	}
}

func TestSort(t *testing.T) {
	testCases := []struct {
		sortOrder         v1alpha1.SortOrder
		podRunnerPairList podRunnerPairList
		podRunnerPair     []podRunnerPair
	}{
		{v1alpha1.LeastRecent, from(&podList, runners), []podRunnerPair{
			{podList.Items[0], *runners[0]},
			{podList.Items[1], *runners[1]}}},
		{v1alpha1.MostRecent, from(&podList, runners), []podRunnerPair{
			{podList.Items[1], *runners[1]},
			{podList.Items[0], *runners[0]}}},
	}

	for _, tc := range testCases {
		podList := tc.podRunnerPairList.getIdles(tc.sortOrder, time.Duration(0))
		assert.Equal(t, tc.podRunnerPair, podList)
	}
}
