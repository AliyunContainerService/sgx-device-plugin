package main

import (
	"flag"
	"strings"
	"syscall"

	"github.com/fsnotify/fsnotify"
	"k8s.io/klog"
	devicepluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	deviceplugin "github.com/AliyunContainerService/sgx-device-plugin/pkg/device_plugin"
	"github.com/AliyunContainerService/sgx-device-plugin/pkg/sgx"
	"github.com/AliyunContainerService/sgx-device-plugin/pkg/utils"
)

func init() {
	flag.BoolVar(&sgx.EnableAESMSocketAttach, "enable-aesm-socket-attach", false, "Enables attachment of AESM service socket for sgx1")
}

func main() {
	flag.Parse()

	if len(sgx.GetDevices()) == 0 {
		panic("No Device Found.")
	}

	klog.Infof("Start watching device plugin socket directory of kubelet ...")
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
			if strings.HasSuffix(event.Name, ".sock") || strings.HasSuffix(event.Name, ".socket") {
				klog.Infof("Event: name - %s, op - %s", event.Name, event.String())
			}

			if event.Name == devicepluginapi.KubeletSocket && event.Op&fsnotify.Create == fsnotify.Create {
				klog.Infof("Inotify: %s created, restarting ...", devicepluginapi.KubeletSocket)
				restart = true
			}

			if event.Name == deviceplugin.ServerSock && event.Op&fsnotify.Remove == fsnotify.Remove {
				klog.Infof("Inotify: %s removed, restarting ...", deviceplugin.ServerSock)
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
