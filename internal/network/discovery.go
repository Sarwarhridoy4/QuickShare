package network

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/Sarwarhridoy4/QuickShare/internal/utils"
)

const (
	DiscoveryPort     = 9998
	BroadcastInterval = 2 * time.Second
	DeviceTimeout     = 10 * time.Second
)

// Device represents a discovered device on the network
type Device struct {
	Name       string    `json:"name"`
	IP         string    `json:"ip"`
	Port       int       `json:"port"`
	LastSeen   time.Time `json:"last_seen"`
	DeviceType string    `json:"device_type"` // desktop, mobile, etc.
}

// DiscoveryService handles device discovery on the local network
type DiscoveryService struct {
	localDevice    Device
	discoveredDevs map[string]*Device
	updateCallback func([]*Device)
	stopChan       chan bool
	conn           *net.UDPConn
}

// NewDiscoveryService creates a new discovery service
func NewDiscoveryService(deviceName string, callback func([]*Device)) *DiscoveryService {
	localIP := GetLocalIP()

	return &DiscoveryService{
		localDevice: Device{
			Name:       deviceName,
			IP:         localIP,
			Port:       Port,
			DeviceType: getDeviceType(),
			LastSeen:   time.Now(),
		},
		discoveredDevs: make(map[string]*Device),
		updateCallback: callback,
		stopChan:       make(chan bool),
	}
}

// Start begins the discovery process
func (ds *DiscoveryService) Start() error {
	utils.Log("Starting discovery service...")

	// Setup UDP listener for broadcasts
	addr := net.UDPAddr{
		Port: DiscoveryPort,
		IP:   net.ParseIP("0.0.0.0"),
	}

	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		return fmt.Errorf("failed to start UDP listener: %w", err)
	}
	ds.conn = conn

	// Start broadcast sender
	go ds.broadcastPresence()

	// Start broadcast receiver
	go ds.listenForDevices()

	// Start cleanup routine
	go ds.cleanupStaleDevices()

	return nil
}

// broadcastPresence sends periodic broadcast messages
func (ds *DiscoveryService) broadcastPresence() {
	ticker := time.NewTicker(BroadcastInterval)
	defer ticker.Stop()

	broadcastAddr := &net.UDPAddr{
		IP:   net.IPv4(255, 255, 255, 255),
		Port: DiscoveryPort,
	}

	for {
		select {
		case <-ticker.C:
			data, err := json.Marshal(ds.localDevice)
			if err != nil {
				utils.LogError("Failed to marshal device info", err)
				continue
			}

			// Send broadcast
			conn, err := net.DialUDP("udp", nil, broadcastAddr)
			if err != nil {
				utils.LogError("Failed to create broadcast connection", err)
				continue
			}

			_, err = conn.Write(data)
			if err != nil {
				utils.LogError("Failed to send broadcast", err)
			}
			conn.Close()

		case <-ds.stopChan:
			return
		}
	}
}

// listenForDevices receives broadcast messages from other devices
func (ds *DiscoveryService) listenForDevices() {
	buffer := make([]byte, 1024)

	for {
		select {
		case <-ds.stopChan:
			return
		default:
			ds.conn.SetReadDeadline(time.Now().Add(1 * time.Second))
			n, _, err := ds.conn.ReadFromUDP(buffer)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				if isExpectedNetCloseError(err) {
					return
				}
				utils.LogError("Error reading UDP packet", err)
				continue
			}

			var device Device
			if err := json.Unmarshal(buffer[:n], &device); err != nil {
				utils.LogError("Failed to unmarshal device info", err)
				continue
			}

			// Ignore own broadcasts
			if device.IP == ds.localDevice.IP {
				continue
			}

			// Update device list
			device.LastSeen = time.Now()
			ds.discoveredDevs[device.IP] = &device

			utils.Log(fmt.Sprintf("Discovered device: %s (%s)", device.Name, device.IP))

			// Notify callback
			if ds.updateCallback != nil {
				ds.updateCallback(ds.GetActiveDevices())
			}
		}
	}
}

// cleanupStaleDevices removes devices that haven't been seen recently
func (ds *DiscoveryService) cleanupStaleDevices() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			now := time.Now()
			updated := false

			for ip, device := range ds.discoveredDevs {
				if now.Sub(device.LastSeen) > DeviceTimeout {
					delete(ds.discoveredDevs, ip)
					utils.Log(fmt.Sprintf("Device removed: %s (%s)", device.Name, ip))
					updated = true
				}
			}

			if updated && ds.updateCallback != nil {
				ds.updateCallback(ds.GetActiveDevices())
			}

		case <-ds.stopChan:
			return
		}
	}
}

// GetActiveDevices returns a list of currently active devices
func (ds *DiscoveryService) GetActiveDevices() []*Device {
	devices := make([]*Device, 0, len(ds.discoveredDevs))
	for _, device := range ds.discoveredDevs {
		devices = append(devices, device)
	}
	return devices
}

// Stop stops the discovery service
func (ds *DiscoveryService) Stop() {
	select {
	case <-ds.stopChan:
		// Already closed.
	default:
		close(ds.stopChan)
	}
	if ds.conn != nil {
		ds.conn.Close()
	}
	utils.Log("Discovery service stopped")
}

func getDeviceType() string {
	// Could be enhanced to detect actual device type
	return "desktop"
}
