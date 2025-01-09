//go:build darwin
// +build darwin

package disk

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/v3/internal/common"
	"golang.org/x/sys/unix"
)

// PartitionsWithContext returns disk partition.
// 'all' argument is ignored, see: https://github.com/giampaolo/psutil/issues/906
func PartitionsWithContext(ctx context.Context, all bool) ([]PartitionStat, error) {
	var ret []PartitionStat

	count, err := unix.Getfsstat(nil, unix.MNT_WAIT)
	if err != nil {
		return ret, err
	}
	fs := make([]unix.Statfs_t, count)
	if _, err = unix.Getfsstat(fs, unix.MNT_WAIT); err != nil {
		return ret, err
	}
	for _, stat := range fs {
		opts := []string{"rw"}
		if stat.Flags&unix.MNT_RDONLY != 0 {
			opts = []string{"ro"}
		}
		if stat.Flags&unix.MNT_SYNCHRONOUS != 0 {
			opts = append(opts, "sync")
		}
		if stat.Flags&unix.MNT_NOEXEC != 0 {
			opts = append(opts, "noexec")
		}
		if stat.Flags&unix.MNT_NOSUID != 0 {
			opts = append(opts, "nosuid")
		}
		if stat.Flags&unix.MNT_UNION != 0 {
			opts = append(opts, "union")
		}
		if stat.Flags&unix.MNT_ASYNC != 0 {
			opts = append(opts, "async")
		}
		if stat.Flags&unix.MNT_DONTBROWSE != 0 {
			opts = append(opts, "nobrowse")
		}
		if stat.Flags&unix.MNT_AUTOMOUNTED != 0 {
			opts = append(opts, "automounted")
		}
		if stat.Flags&unix.MNT_JOURNALED != 0 {
			opts = append(opts, "journaled")
		}
		if stat.Flags&unix.MNT_MULTILABEL != 0 {
			opts = append(opts, "multilabel")
		}
		if stat.Flags&unix.MNT_NOATIME != 0 {
			opts = append(opts, "noatime")
		}
		if stat.Flags&unix.MNT_NODEV != 0 {
			opts = append(opts, "nodev")
		}
		d := PartitionStat{
			Device:     common.ByteToString(stat.Mntfromname[:]),
			Mountpoint: common.ByteToString(stat.Mntonname[:]),
			Fstype:     common.ByteToString(stat.Fstypename[:]),
			Opts:       opts,
		}

		ret = append(ret, d)
	}

	return ret, nil
}

func getFsType(stat unix.Statfs_t) string {
	return common.ByteToString(stat.Fstypename[:])
}

func SerialNumberWithContext(ctx context.Context, name string) (string, error) {
	return "", common.ErrNotImplementedError
}

func LabelWithContext(ctx context.Context, name string) (string, error) {
	return "", common.ErrNotImplementedError
}

func UsageWithContext(ctx context.Context, path string) (*UsageStat, error) {
	cmd := exec.Command("/bin/df", "-ki", path)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("error executing df command: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("unexpected df output format")
	}

	// Skip the header line and process the data line
	fields := strings.Fields(lines[1])
	if len(fields) < 9 {
		return nil, fmt.Errorf("unexpected number of fields in df output")
	}

	// Parse values
	total, err := parseUint64(fields[1])
	if err != nil {
		return nil, fmt.Errorf("error parsing total blocks: %v", err)
	}
	used, err := parseUint64(fields[2])
	if err != nil {
		return nil, fmt.Errorf("error parsing used blocks: %v", err)
	}
	free, err := parseUint64(fields[3])
	if err != nil {
		return nil, fmt.Errorf("error parsing available blocks: %v", err)
	}
	inodesUsed, err := parseUint64(fields[5])
	if err != nil {
		return nil, fmt.Errorf("error parsing iused: %v", err)
	}
	inodesFree, err := parseUint64(fields[6])
	if err != nil {
		return nil, fmt.Errorf("error parsing ifree: %v", err)
	}

	// Calculate percentages
	usedPercent := float64(used) / float64(total) * 100
	inodesTotal := inodesUsed + inodesFree
	inodesUsedPercent := float64(inodesUsed) / float64(inodesTotal) * 100

	// Create UsageStat object
	us := &UsageStat{
		Path:              fields[8],
		Fstype:            fields[0],
		Total:             total * 1024,
		Free:              free * 1024,
		Used:              used * 1024,
		UsedPercent:       usedPercent,
		InodesTotal:       inodesTotal,
		InodesUsed:        inodesUsed,
		InodesFree:        inodesFree,
		InodesUsedPercent: inodesUsedPercent,
	}

	return us, nil
}

func parseUint64(s string) (uint64, error) {
	return strconv.ParseUint(s, 10, 64)
}
