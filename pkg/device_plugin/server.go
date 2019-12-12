package deviceplugin

import (
	"errors"
	"net"
	"os"
	"path"
	"strings"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"k8s.io/klog"
	devicepluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"

	"github.com/AliyunContainerService/sgx-device-plugin/pkg/sgx"
)

const (
	vendor = "alibabacloud.com"
	// ResourceNameSGX is resource name registered to kubelet.
	ResourceNameSGX = vendor + "/sgx_epc_MiB"

	serverSock             = devicepluginapi.DevicePluginPath + "/sgx.sock"
	envDisableHealthChecks = "DP_DISABLE_HEALTHCHECKS"
	allHealthChecks        = "xids"
)

// NewSGXDevicePlugin returns an initialized SGXDevicePlugin
func NewSGXDevicePlugin() (*SGXDevicePlugin, error) {
	drivers := sgx.AllDeviceDrivers()
	if _, ok := drivers["/dev/isgx"]; !ok {
		return nil, errors.New("/dev/isgx not found")
	}

	devs := sgx.GetDevices()
	if len(devs) == 0 {
		return nil, errors.New("empty devices list")
	}

	return &SGXDevicePlugin{
		devs:   devs,
		socket: serverSock,

		stop:   make(chan interface{}),
		health: make(chan *devicepluginapi.Device),
	}, nil
}

// dial establishes the gRPC communication with the registered device plugin.
func dial(unixSocketPath string, timeout time.Duration) (*grpc.ClientConn, error) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, timeout)

	c, err := grpc.DialContext(ctx,
		unixSocketPath,
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithContextDialer(func(i context.Context, addr string) (conn net.Conn, e error) {
			return net.DialTimeout("unix", addr, timeout)
		}),
	)

	if err != nil {
		cancel()
		return nil, err
	}

	return c, nil
}

// Start starts the gRPC server of the device plugin
func (m *SGXDevicePlugin) Start() error {
	err := m.cleanup()
	if err != nil {
		return err
	}

	sock, err := net.Listen("unix", m.socket)
	if err != nil {
		return err
	}

	m.server = grpc.NewServer([]grpc.ServerOption{}...)
	devicepluginapi.RegisterDevicePluginServer(m.server, m)

	go func() {
		lastCrashTime := time.Now()
		restartCount := 0
		for {
			klog.Infof("Starting GRPC server")
			err := m.server.Serve(sock)
			if err != nil {
				klog.Errorf("GRPC server crashed with error: %v", err)
			}
			// restart if it has not been too often
			// i.e. if server has crashed more than 5 times and it didn't last more than one hour each time
			if restartCount > 5 {
				// quit
				klog.Fatalf("GRPC server has repeatedly crashed recently. Quitting")
			}
			timeSinceLastCrash := time.Since(lastCrashTime).Seconds()
			lastCrashTime = time.Now()
			if timeSinceLastCrash > 3600 {
				// it has been one hour since the last crash.. reset the count
				// to reflect on the frequency
				restartCount = 1
			} else {
				restartCount++
			}
		}
	}()

	// Wait for server to start by launching a blocking connection
	conn, err := dial(m.socket, 5*time.Second)
	if err != nil {
		return err
	}
	conn.Close()

	go m.healthcheck()

	return nil
}

// Stop stops the gRPC server
func (m *SGXDevicePlugin) Stop() error {
	if m.server == nil {
		return nil
	}

	m.server.Stop()
	m.server = nil
	close(m.stop)

	return m.cleanup()
}

// Register registers the device plugin for the given resourceName with Kubelet.
func (m *SGXDevicePlugin) Register(kubeletEndpoint, resourceName string) error {
	conn, err := dial(kubeletEndpoint, 5*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()

	client := devicepluginapi.NewRegistrationClient(conn)
	reqt := &devicepluginapi.RegisterRequest{
		Version:      devicepluginapi.Version,
		Endpoint:     path.Base(m.socket),
		ResourceName: ResourceNameSGX,
	}

	_, err = client.Register(context.Background(), reqt)
	if err != nil {
		return err
	}
	return nil
}

func (m *SGXDevicePlugin) unhealthy(dev *devicepluginapi.Device) {
	m.health <- dev
}

func (m *SGXDevicePlugin) cleanup() error {
	if err := os.Remove(m.socket); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (m *SGXDevicePlugin) healthcheck() {
	disableHealthChecks := strings.ToLower(os.Getenv(envDisableHealthChecks))
	if disableHealthChecks == "all" {
		disableHealthChecks = allHealthChecks
	}

	ctx, cancel := context.WithCancel(context.Background())

	var xids chan *devicepluginapi.Device
	if !strings.Contains(disableHealthChecks, "xids") {
		xids = make(chan *devicepluginapi.Device)
		go sgx.WatchXIDs(ctx, m.devs, xids)
	}

	for {
		select {
		case <-m.stop:
			cancel()
			return
		case dev := <-xids:
			m.unhealthy(dev)
		}
	}
}

// Serve starts the gRPC server and register the device plugin to Kubelet
func (m *SGXDevicePlugin) Serve() error {
	err := m.Start()
	if err != nil {
		klog.Errorf("Could not start device plugin: %s", err)
		return err
	}
	klog.Infof("Starting to serve on %s", m.socket)

	err = m.Register(devicepluginapi.KubeletSocket, ResourceNameSGX)
	if err != nil {
		klog.Errorf("Could not register device plugin: %s", err)
		_ = m.Stop()
		return err
	}
	klog.Infof("Registered device plugin with Kubelet")

	return nil
}
