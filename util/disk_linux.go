package util

import (
	"golang.org/x/sys/unix"
)

func DiskUsage(dir string) (freeBytes uint64, totalBytes uint64, err error) {
	var statfs = unix.Statfs_t{}

	err = unix.Statfs(dir, &statfs)

	if err != nil {
		return 0, 0, err
	}

	freeBytes = statfs.Bfree * uint64(statfs.Bsize)
	totalBytes = statfs.Blocks * uint64(statfs.Bsize)

	return freeBytes, totalBytes, nil
}
