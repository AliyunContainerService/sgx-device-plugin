package sgx

import (
	"os"
	"strconv"
	"sync"

	"golang.org/x/net/context"
	devicepluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"
)

var initOnce = &sync.Once{}

var allDevices = map[string]bool{
	"/dev/isgx": false, // required
	"/dev/gsgx": false, // optional
}

func AllDevices() map[string]bool {
	ret := make(map[string]bool)
	for k, v := range allDevices {
		ret[k] = v
	}
	return ret
}

func init() {
	initOnce.Do(func() {
		// Detecting devices.
		for dev, _ := range allDevices {
			if fi, err := os.Stat(dev); err == nil && !fi.IsDir() {
				allDevices[dev] = true
			}
		}

		// isgx is required.
		if !allDevices["/dev/isgx"] {
			panic("/dev/isgx not found")
		}

	})
}

const (
	sgxCpuidEpcSections = 2
	sgxMaxEpcSections   = 8
	cpuidSgxResources   = 0x12
)

type SgxEpcSection struct {
	PhysicalAddress uint64
	Size            uint64
}

func cpuid_low(leaf, subLeaf uint32) (eax, ebx, ecx, edx uint32)

func GetEPCSections() []SgxEpcSection {
	sections := []SgxEpcSection{}

	for i := 0; i < sgxMaxEpcSections; i++ {
		eax, ebx, ecx, edx := cpuid_low(cpuidSgxResources, uint32(sgxCpuidEpcSections+i))

		if (eax & 0xf) == 0x0 {
			break
		}

		pa := ((uint64)(ebx&0xfffff) << 32) + (uint64)(eax&0xfffff000)
		sz := ((uint64)(edx&0xfffff) << 32) + (uint64)(ecx&0xfffff000)

		sections = append(sections, SgxEpcSection{pa, sz})
	}

	return sections
}

func GetEPCSize() uint64 {
	sections := GetEPCSections()

	var epcSize uint64 = 0

	for _, s := range sections {
		epcSize += s.Size
	}

	return epcSize
}

func GetDevices() []*devicepluginapi.Device {
	sizeMB := GetEPCSize() / 1024 / 1024
	devs := make([]*devicepluginapi.Device, 0, sizeMB)

	for i := uint64(0); i < sizeMB; i++ {
		devs = append(devs, &devicepluginapi.Device{
			ID:     strconv.FormatUint(i, 10),
			Health: devicepluginapi.Healthy,
		})
	}

	return devs
}

func DeviceExists(devs []*devicepluginapi.Device, id string) bool {
	for _, d := range devs {
		if d.ID == id {
			return true
		}
	}
	return false
}

func WatchXIDs(ctx context.Context, devs []*devicepluginapi.Device, xids chan<- *devicepluginapi.Device) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}
