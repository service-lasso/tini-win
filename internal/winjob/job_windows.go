//go:build windows

package winjob

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

type Handle = windows.Handle

func CreateAndAssign(pid uint32, killOnClose bool) (Handle, error) {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return 0, fmt.Errorf("CreateJobObject: %w", err)
	}

	if killOnClose {
		var info windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION
		info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
		_, err = windows.SetInformationJobObject(
			job,
			windows.JobObjectExtendedLimitInformation,
			uintptr(unsafe.Pointer(&info)),
			uint32(unsafe.Sizeof(info)),
		)
		if err != nil {
			windows.CloseHandle(job)
			return 0, fmt.Errorf("SetInformationJobObject: %w", err)
		}
	}

	proc, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE|windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		windows.CloseHandle(job)
		return 0, fmt.Errorf("OpenProcess: %w", err)
	}
	defer windows.CloseHandle(proc)

	if err := windows.AssignProcessToJobObject(job, proc); err != nil {
		windows.CloseHandle(job)
		return 0, fmt.Errorf("AssignProcessToJobObject: %w", err)
	}

	return job, nil
}

func Terminate(job Handle, code uint32) error {
	return windows.TerminateJobObject(job, code)
}

func Close(job Handle) {
	_ = windows.CloseHandle(job)
}
