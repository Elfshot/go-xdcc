package util

import (
	"golang.org/x/sys/windows"
)

func DiskUsage(dir string) (freeBytes uint64, totalBytes uint64, err error) {
	dirr := windows.StringToUTF16Ptr(dir)
	var freeBytesAvailableToCaller, totalNumberOfBytes, totalNumberOfFreeBytes uint64

	err = windows.GetDiskFreeSpaceEx(dirr, &freeBytesAvailableToCaller, &totalNumberOfBytes, &totalNumberOfFreeBytes)

	if err != nil {
		return 0, 0, err
	}
	return freeBytesAvailableToCaller, totalNumberOfBytes, nil
}
