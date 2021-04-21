package sgx

import (
	"os"
	"strconv"
	"sync"
	"time"

	"golang.org/x/net/context"
	"k8s.io/klog"
	devicepluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

var EnableAESMSocketAttach bool
var AESMSocketDir string = "/var/run/aesmd"

var initOnce = &sync.Once{}

var allMountPoints = map[string]bool{
	AESMSocketDir: false, // optional
}

// AllMountPoints lists all mount points.
func AllMountPoints() map[string]bool {
	ret := make(map[string]bool)
	for k, v := range allMountPoints {
		ret[k] = v
	}
	return ret
}

var allDeviceDrivers = map[string]bool{
	"/dev/isgx": false, // required out-of-tree sgx driver
	"/dev/sgx":  false, // alternative in-tree sgx driver
	"/dev/gsgx": false, // optional
}

// AllDeviceDrivers lists all device drivers.
func AllDeviceDrivers() map[string]bool {
	ret := make(map[string]bool)
	for k, v := range allDeviceDrivers {
		ret[k] = v
	}
	return ret
}

func init() {
	initOnce.Do(func() {
		// Detecting mount points.
		klog.Infof("Detecting mount points ...")
		for mp := range allMountPoints {
			if fi, err := os.Stat(mp); err == nil && fi.IsDir() {
				allMountPoints[mp] = true
				klog.Infof("\tFound mount point: %s", mp)
			}
		}
		// Detecting device drivers.
		klog.Infof("Detecting device drivers ...")
		for driver := range allDeviceDrivers {
			if fi, err := os.Stat(driver); err == nil && !fi.IsDir() {
				allDeviceDrivers[driver] = true
				klog.Infof("\tFound device driver: %s", driver)
			}
		}
	})
}

const (
	sgxCpuidEpcSections = 2
	sgxMaxEpcSections   = 8
	cpuidSgxResources   = 0x12
)

// EPCSection - ECP Section(Bank).
type EPCSection struct {
	PhysicalAddress uint64
	Size            uint64
}

func cpuidLow(leaf, subLeaf uint32) (eax, ebx, ecx, edx uint32)

// GetEPCSections lists all EPC sections.
func GetEPCSections() []EPCSection {
	sections := []EPCSection{}

	for i := 0; i < sgxMaxEpcSections; i++ {
		eax, ebx, ecx, edx := cpuidLow(cpuidSgxResources, uint32(sgxCpuidEpcSections+i))

		if (eax & 0xf) == 0x0 {
			break
		}

		pa := ((uint64)(ebx&0xfffff) << 32) + (uint64)(eax&0xfffff000)
		sz := ((uint64)(edx&0xfffff) << 32) + (uint64)(ecx&0xfffff000)

		sections = append(sections, EPCSection{pa, sz})
	}

	return sections
}

// GetEPCSize returns total EPC size.
func GetEPCSize() uint64 {
	sections := GetEPCSections()

	var epcSize uint64

	for _, s := range sections {
		epcSize += s.Size
	}

	return epcSize
}

// GetDevices divides EPC into many virtual devices, each device's ecp memory is 1MiB.
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

// DeviceExists check device existence by id.
func DeviceExists(devs []*devicepluginapi.Device, id string) bool {
	for _, d := range devs {
		if d.ID == id {
			return true
		}
	}
	return false
}

// WatchXIDs is used for device health-check.
func WatchXIDs(ctx context.Context, devs []*devicepluginapi.Device, xids chan<- *devicepluginapi.Device) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			time.Sleep(time.Second)
		}
	}
}
