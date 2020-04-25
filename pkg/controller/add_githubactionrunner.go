package controller

import (
	"github.com/evryfs/github-actions-runner-operator/pkg/controller/githubactionrunner"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, githubactionrunner.Add)
}
