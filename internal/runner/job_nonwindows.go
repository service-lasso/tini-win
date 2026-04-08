//go:build !windows

package runner

type jobHandle struct{}

func createAndAssignJobObject(pid uint32, killTree bool) (jobHandle, error) { return jobHandle{}, nil }
func terminateJobObject(job jobHandle, code uint32) error                   { return nil }
func closeJobObject(job jobHandle)                                          {}
