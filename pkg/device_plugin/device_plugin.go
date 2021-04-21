package deviceplugin

import (
	"fmt"
	"strings"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"k8s.io/klog"
	devicepluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	"github.com/AliyunContainerService/sgx-device-plugin/pkg/sgx"
)

var _ devicepluginapi.DevicePluginServer = &SGXDevicePlugin{}

// SGXDevicePlugin implements the Kubernetes device plugin API: DevicePluginServer.
type SGXDevicePlugin struct {
	devs   []*devicepluginapi.Device
	socket string

	stop   chan interface{}
	health chan *devicepluginapi.Device

	server *grpc.Server
}

// GetDevicePluginOptions implements DevicePluginServer interface.
// We just do nothing here.
func (m *SGXDevicePlugin) GetDevicePluginOptions(context.Context, *devicepluginapi.Empty) (*devicepluginapi.DevicePluginOptions, error) {
	return &devicepluginapi.DevicePluginOptions{}, nil
}

// ListAndWatch lists devices and update that list according to the health status.
// ListAndWatch implements DevicePluginServer interface.
func (m *SGXDevicePlugin) ListAndWatch(e *devicepluginapi.Empty, s devicepluginapi.DevicePlugin_ListAndWatchServer) error {
	if err := s.Send(&devicepluginapi.ListAndWatchResponse{Devices: m.devs}); err != nil {
		klog.Errorf("Send ListAndWatchResponse error: %v", err)
	}

	for {
		select {
		case <-m.stop:
			return nil
		case d := <-m.health:
			// FIXME: there is no way to recover from the Unhealthy state.
			d.Health = devicepluginapi.Unhealthy
			if err := s.Send(&devicepluginapi.ListAndWatchResponse{Devices: m.devs}); err != nil {
				klog.Errorf("Send ListAndWatchResponse error: %v", err)
			}
		}
	}
}

// GetPreferredAllocation implements DevicePluginServer interface.
func (m *SGXDevicePlugin) GetPreferredAllocation(ctx context.Context, req *devicepluginapi.PreferredAllocationRequest) (*devicepluginapi.PreferredAllocationResponse, error) {
	return &devicepluginapi.PreferredAllocationResponse{}, nil
}

// Allocate which return list of devices.
// Allocate implements DevicePluginServer interface.
func (m *SGXDevicePlugin) Allocate(ctx context.Context, reqs *devicepluginapi.AllocateRequest) (*devicepluginapi.AllocateResponse, error) {
	var mounts []*devicepluginapi.Mount
	var devices []*devicepluginapi.DeviceSpec

	if sgx.EnableAESMSocketAttach {
		for path, exist := range sgx.AllMountPoints() {
			if path != sgx.AESMSocketDir {
				continue
			}
			if exist {
				mounts = append(mounts, &devicepluginapi.Mount{
					ContainerPath: path,
					HostPath:      path,
					ReadOnly:      true,
				})
			} else {
				klog.Warningf("WARNING: Mount point %s not found", path)
			}
		}
	}

	for dev, exist := range sgx.AllDeviceDrivers() {
		if exist {
			devices = append(devices, &devicepluginapi.DeviceSpec{
				ContainerPath: dev,
				HostPath:      dev,
				Permissions:   "rw",
			})
		} else {
			klog.Warningf("WARNING: Device %s not found", dev)
		}
	}

	responses := devicepluginapi.AllocateResponse{}
	for _, req := range reqs.ContainerRequests {
		response := devicepluginapi.ContainerAllocateResponse{
			Envs: map[string]string{
				"SGX_VISIBLE_DEVICES": strings.Join(req.DevicesIDs, ","),
			},
			Devices: devices,
			Mounts:  mounts,
		}

		klog.Infof("[Allocate] %s", req.String())

		for _, id := range req.DevicesIDs {
			if !sgx.DeviceExists(m.devs, id) {
				return nil, fmt.Errorf("invalid allocation request: unknown device: %s", id)
			}
		}

		responses.ContainerResponses = append(responses.ContainerResponses, &response)
	}

	return &responses, nil
}

// PreStartContainer implements DevicePluginServer interface.
// We just do nothing here.
func (m *SGXDevicePlugin) PreStartContainer(context.Context, *devicepluginapi.PreStartContainerRequest) (*devicepluginapi.PreStartContainerResponse, error) {
	return &devicepluginapi.PreStartContainerResponse{}, nil
}
