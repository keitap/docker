// +build windows

package devices

import (
	"os"
	"syscall"
)

// Given the path to a device and it's cgroup_permissions(which cannot be easilly queried) look up the information about a linux device and return that information as a Device struct.
func DeviceFromPath(path, permissions string) (*configs.Device, error) {
	fileInfo, err := osLstat(path)
	if err != nil {
		return nil, err
	}
	var (
		devType                rune
		mode                   = fileInfo.Mode()
		fileModePermissionBits = os.FileMode.Perm(mode)
	)
	switch {
	case mode&os.ModeDevice == 0:
		return nil, ErrNotADevice
	case mode&os.ModeCharDevice != 0:
		fileModePermissionBits |= syscall.S_IFCHR
		devType = 'c'
	default:
		fileModePermissionBits |= syscall.S_IFBLK
		devType = 'b'
	}
	//stat_t, ok := fileInfo.Sys().(*syscall.Stat_t)
	//if !ok {
	//	return nil, fmt.Errorf("cannot determine the device number for device %s", path)
	//}
	//devNumber := int(stat_t.Rdev)
	return &configs.Device{
		Type:        devType,
		Path:        path,
		Major:       Major(devNumber),
		Minor:       Minor(devNumber),
		Permissions: permissions,
		FileMode:    fileModePermissionBits,
		Uid:         stat_t.Uid,
		Gid:         stat_t.Gid,
	}, nil
}
