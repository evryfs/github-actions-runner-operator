package controllers

import (
	"github.com/google/go-github/v33/github"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"testing"
)

func TestSomething(t *testing.T) {
	podList := v1.PodList{
		TypeMeta: metav1.TypeMeta{},
		ListMeta: metav1.ListMeta{},
		Items: []v1.Pod{
			{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "name1",
				},
				Spec:   v1.PodSpec{},
				Status: v1.PodStatus{},
			},
			{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "name2",
				},
				Spec:   v1.PodSpec{},
				Status: v1.PodStatus{},
			},
		},
	}

	runners := []*github.Runner{
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

	podRunnerPairList := from(&podList, runners)
	assert.True(t, podRunnerPairList.inSync())
}
