package network

import (
	"fmt"
	"net"
)

// GetLocalIP returns the local IP address of the device
func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "Unknown"
	}
	
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	
	return "127.0.0.1"
}

// GetAllNetworkInterfaces returns all available network interfaces
func GetAllNetworkInterfaces() ([]NetworkInterface, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get network interfaces: %w", err)
	}
	
	var result []NetworkInterface
	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					result = append(result, NetworkInterface{
						Name:       iface.Name,
						IP:         ipnet.IP.String(),
						MACAddress: iface.HardwareAddr.String(),
					})
				}
			}
		}
	}
	
	return result, nil
}

// NetworkInterface represents a network interface
type NetworkInterface struct {
	Name       string
	IP         string
	MACAddress string
}

// ValidateIP checks if an IP address is valid
func ValidateIP(ip string) bool {
	parsedIP := net.ParseIP(ip)
	return parsedIP != nil
}

// IsPortAvailable checks if a port is available
func IsPortAvailable(port int) bool {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	listener.Close()
	return true
}