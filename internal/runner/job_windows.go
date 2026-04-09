//go:build windows

package runner

import "github.com/service-lasso/tini-win/internal/winjob"

type jobHandle = winjob.Handle

func createAndAssignJobObject(pid uint32, killTree bool, allowBreakaway bool) (jobHandle, error) {
	return winjob.CreateAndAssign(pid, killTree, allowBreakaway)
}

func terminateJobObject(job jobHandle, code uint32) error {
	return winjob.Terminate(job, code)
}

func closeJobObject(job jobHandle) {
	winjob.Close(job)
}
