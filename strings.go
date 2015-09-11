package gorpc

import "fmt"

func (device *Device) String() string {
	return fmt.Sprintf("%s-%s-%s", device.ID, device.OS, device.OSVersion)
}
