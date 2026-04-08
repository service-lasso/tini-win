//go:build !windows

package winjob

type Handle uintptr

func CreateAndAssign(pid uint32, killOnClose bool) (Handle, error) { return 0, nil }
func Terminate(job Handle, code uint32) error                      { return nil }
func Close(job Handle)                                             {}
