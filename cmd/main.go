package main

import (
	"syscall"

	"github.com/fsnotify/fsnotify"
	"k8s.io/klog"
	devicepluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"

	"github.com/AliyunContainerService/sgx-device-plugin/cmd/app"
	"github.com/AliyunContainerService/sgx-device-plugin/pkg/sgx"
)

func main() {
	klog.Infof("Detecting SGX devices ...")
	if len(sgx.GetDevices()) == 0 {
		panic("No Device Found.")
	}

	klog.Infof("Start watching kubelet.socket ...")
	watcher, err := sgx.NewFSWatcher(devicepluginapi.DevicePluginPath)
	if err != nil {
		panic(err)
	}
	defer watcher.Close()

	sigs := sgx.NewOSWatcher(syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	restart := true
	var devicePlugin *app.SgxDevicePlugin

L:
	for {
		if restart {
			if devicePlugin != nil {
				devicePlugin.Stop()
			}

			devicePlugin = app.NewSgxDevicePlugin()
			if err := devicePlugin.Serve(); err != nil {
				klog.Infof("Could not contact Kubelet, retrying. Did you enable the device plugin feature gate?")
			} else {
				restart = false
			}
		}

		select {
		case event := <-watcher.Events:
			if event.Name == devicepluginapi.KubeletSocket && event.Op&fsnotify.Create == fsnotify.Create {
				klog.Infof("inotify: %s created, restarting.", devicepluginapi.KubeletSocket)
				restart = true
			}

		case err := <-watcher.Errors:
			klog.Infof("inotify: %s", err)

		case s := <-sigs:
			switch s {
			case syscall.SIGHUP:
				klog.Infof("Received SIGHUP, restarting.")
				restart = true
			default:
				klog.Infof("Received signal \"%v\", shutting down.", s)
				devicePlugin.Stop()
				break L
			}
		}
	}
}
