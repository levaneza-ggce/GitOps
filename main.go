package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"encoding/json"

	"golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v2"
)

type VLAN struct {
	ID   int    `yaml:"id"`
	Name string `yaml:"name"`
}

type VLANConfig struct {
	VLANs []VLAN `yaml:"vlans"`
}

type DeviceVault struct {
	Host         string `json:"host"`
	User         string `json:"user"`
	Password     string `json:"password"`
	EnableSecret string `json:"enable_secret"`
	Port         int    `json:"port"`
}

func readYAMLConfig(filename string) (*VLANConfig, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var config VLANConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func readDeviceVault(filename string) (*DeviceVault, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var vault DeviceVault
	if err := json.Unmarshal(data, &vault); err != nil {
		return nil, err
	}
	return &vault, nil
}

func sshConnectAndConfigVLANs(vault *DeviceVault, vlans []VLAN) error {
	config := &ssh.ClientConfig{
		User: vault.User,
		Auth: []ssh.AuthMethod{
			ssh.Password(vault.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	address := fmt.Sprintf("%s:%d", vault.Host, vault.Port)
	client, err := ssh.Dial("tcp", address, config)
	if err != nil {
		return err
	}
	defer client.Close()

	for _, vlan := range vlans {
		session, err := client.NewSession()
		if err != nil {
			return err
		}
		defer session.Close()
		cmd := fmt.Sprintf("enable\n%s\nconfigure terminal\nvlan %d\nname %s\nend\nwrite memory\n", vault.EnableSecret, vlan.ID, vlan.Name)
		if err := session.Run(cmd); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	if len(os.Args) < 3 {
		log.Fatalf("Usage: %s <device_vault.json> <vlan_config.yaml>", os.Args[0])
	}
	vault, err := readDeviceVault(os.Args[1])
	if err != nil {
		log.Fatalf("Failed to read device vault: %v", err)
	}
	config, err := readYAMLConfig(os.Args[2])
	if err != nil {
		log.Fatalf("Failed to read YAML: %v", err)
	}
	if err := sshConnectAndConfigVLANs(vault, config.VLANs); err != nil {
		log.Fatalf("Failed to configure VLANs: %v", err)
	}
	fmt.Println("VLAN configuration applied successfully.")
}
