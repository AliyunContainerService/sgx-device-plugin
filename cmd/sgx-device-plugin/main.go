package main

import (
	"syscall"

	"github.com/fsnotify/fsnotify"
	"k8s.io/klog"
	devicepluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"

	deviceplugin "github.com/AliyunContainerService/sgx-device-plugin/pkg/device_plugin"
	"github.com/AliyunContainerService/sgx-device-plugin/pkg/sgx"
	"github.com/AliyunContainerService/sgx-device-plugin/pkg/utils"
)

func main() {
	klog.Infof("Detecting SGX devices ...")
	if len(sgx.GetDevices()) == 0 {
		panic("No Device Found.")
	}

	klog.Infof("Start watching kubelet.socket ...")
	watcher, err := utils.NewFSWatcher(devicepluginapi.DevicePluginPath)
	if err != nil {
		panic(err)
	}
	defer watcher.Close()

	sigs := utils.NewOSWatcher(syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	restart := true
	var devicePlugin *deviceplugin.SGXDevicePlugin

L:
	for {
		if restart {
			if devicePlugin != nil {
				if err := devicePlugin.Stop(); err != nil {
					panic(err)
				}
			}

			devicePlugin, err = deviceplugin.NewSGXDevicePlugin()
			if err != nil {
				panic(err)
			}

			if err := devicePlugin.Serve(); err != nil {
				klog.Infof("Could not contact Kubelet, retrying. Did you enable the device plugin feature gate?")
			} else {
				restart = false
			}
		}

		select {
		case event := <-watcher.Events:
			if event.Name == devicepluginapi.KubeletSocket && event.Op&fsnotify.Create == fsnotify.Create {
				klog.Infof("Inotify: %s created, restarting ...", devicepluginapi.KubeletSocket)
				restart = true
			}

		case err := <-watcher.Errors:
			klog.Infof("inotify: %s", err)

		case s := <-sigs:
			switch s {
			case syscall.SIGHUP:
				klog.Infof("Received SIGHUP, restarting ...")
				restart = true
			default:
				klog.Infof("Received signal \"%v\", shutting down.", s)
				if err := devicePlugin.Stop(); err != nil {
					panic(err)
				}
				break L
			}
		}
	}
}
