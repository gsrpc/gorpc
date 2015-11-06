package gorpc

import "fmt"

func (device *Device) String() string {
	return fmt.Sprintf("%s-%s-%s", device.ID, device.OS, device.OSVersion)
}

func (named *NamedService) String() string {
	return fmt.Sprintf("%s:%s-%d:%d", named.NodeName, named.Name, named.DispatchID, named.VNodes)
}
