// A ConfigGenerator that generates a configuration file for HAProxy.
// Restarts HAProxy with the new config in case there are changes.
// It uses domain based routing for HTTP applications.
//
// It requires the following environment variables to be set:
// HAPROXY_BINARY_PATH - The absolute path to the binary of HAProxy
//
// HAPROXY_CONFIG_FILE_PATH - An absolute path where the generated config file will be stored.
//
// HAPROXY_HTTP_PORT - Define the port under which applications available via HTTP will be reachable. This is usually 80.
//
// HAPROXY_OPTIONS_PATH - An absolute path to a file where 'global' options and 'default's of HAProxy are stored. The
//                        content of this file prepended to the rest of the config.
//
// HAPROXY_PID_PATH - An absolute where HAProxy stores its PID.
package haproxy

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/wndhydrnt/proxym/types"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type Config struct {
	BinPath        string
	ConfigFilePath string
	HttpPort       int
	OptionsPath    string
	ProcessPidPath string
}

type HAProxyGenerator struct {
	c *Config
}

// Creates a new HAproxy config file and reloads HAProxy
func (h *HAProxyGenerator) Generate(services []types.Service) {
	currentConfig, _ := readExistingFile(h.c.ConfigFilePath)
	newConfig := h.config(services)

	// No change. Do nothing.
	if currentConfig == newConfig {
		return
	}

	f, err := os.Create(h.c.ConfigFilePath)
	if err != nil {
		log.Printf("haproxy.HAProxyGenerator.Generate: Error opening config file for reading '%s': %s", h.c.ConfigFilePath, err)
		return
	}

	defer f.Close()

	_, err = f.WriteString(newConfig)
	if err != nil {
		log.Printf("haproxy.HAProxyGenerator.Generate: Error writing config file '%s': %s", h.c.ConfigFilePath, err)
		return
	}

	cmdParts := []string{"-f", h.c.ConfigFilePath, "-p", h.c.ProcessPidPath}

	pid, err := readExistingFile(h.c.ProcessPidPath)
	if err == nil {
		cmdParts = append(cmdParts, "-sf")
		cmdParts = append(cmdParts, pid)
	}

	cmd := exec.Command(h.c.BinPath, cmdParts...)
	var cmdErr bytes.Buffer
	cmd.Stderr = &cmdErr

	err = cmd.Run()
	if err != nil {
		log.Printf("haproxy.HAProxyGenerator.Generate: Error starting HAProxy: %s", err)
		log.Printf("haproxy.HAProxyGenerator.Generate: HAProxy Stderr: %s", cmdErr.String())
	}
}

func (h *HAProxyGenerator) config(services []types.Service) string {
	var httpServices []types.Service
	var tcpServices []types.Service

	for _, service := range services {
		if service.Protocol == "tcp" && service.ServicePort == h.c.HttpPort {
			httpServices = append(httpServices, service)
			continue
		}

		if service.Protocol == "tcp" {
			tcpServices = append(tcpServices, service)
		}
		// HAProxy supports TCP only. Ignore any other protocol.
	}

	globalConfig, err := readExistingFile(h.c.OptionsPath + "/global.cfg")
	if err != nil {
		log.Printf("haproxy.HAProxyGenerator.httpConfig: Error reading global config. Stopping HAProxy config generator: %s", err)
	}

	httpConfig := h.httpConfig(httpServices)
	tcpConfig := h.tcpConfig(tcpServices)

	fullConfig := globalConfig + httpConfig + tcpConfig

	return fullConfig
}

func (h *HAProxyGenerator) httpConfig(services []types.Service) string {
	var buffer bytes.Buffer
	aclLines := []string{}
	useBackendLines := []string{}

	buffer.WriteString("\nfrontend http-in\n  bind *:80\n\n")

	for _, service := range services {
		name := h.generateName(service.Id)

		aclLine := fmt.Sprintf("  acl host_%s hdr(host) -i %s", name, service.Domain)
		aclLines = append(aclLines, aclLine)

		useBackendLine := fmt.Sprintf("  use_backend %s_cluster if host_%s", name, name)
		useBackendLines = append(useBackendLines, useBackendLine)
	}

	buffer.WriteString(strings.Join(aclLines, "\n"))
	buffer.WriteString("\n\n")
	buffer.WriteString(strings.Join(useBackendLines, "\n"))
	buffer.WriteString("\n")

	for _, service := range services {
		buffer.WriteString("\n")

		name := h.generateName(service.Id)

		buffer.WriteString(fmt.Sprintf("backend %s_cluster\n", name))

		optionData, err := h.readOptions(service.Domain, service.ServicePort)
		if err != nil {
			log.Printf("haproxy.HAProxyGenerator.httpConfig: Error reading options file: '%s'", err)
		}

		for _, option := range h.sanitizeOptions(optionData) {
			buffer.WriteString(fmt.Sprintf("  %s\n", option))
		}

		buffer.WriteString("\n")

		for i, host := range service.Hosts {
			nodeNumber := i + 1
			buffer.WriteString(fmt.Sprintf("  server node%d %s:%d check\n", nodeNumber, host.Ip, host.Port))
		}
	}

	return buffer.String()
}

func (h *HAProxyGenerator) tcpConfig(services []types.Service) string {
	var buffer bytes.Buffer

	for _, service := range services {
		name := h.generateName(service.Id)
		header := fmt.Sprintf("\nlisten %s :%d\n  mode tcp\n", name, service.ServicePort)
		buffer.WriteString(header)

		optionData, err := h.readOptions(service.Domain, service.ServicePort)
		if err != nil {
			log.Println("haproxy.HAProxyGenerator.tcpConfig: Error reading options file: '%s'", err)
		}

		for _, option := range h.sanitizeOptions(optionData) {
			buffer.WriteString(fmt.Sprintf("  %s\n", option))
		}

		buffer.WriteString("\n")

		for index, host := range service.Hosts {
			nodeNumber := index + 1
			serverLine := fmt.Sprintf("  server node%d %s:%d check\n", nodeNumber, host.Ip, host.Port)
			buffer.WriteString(serverLine)
		}
	}

	return buffer.String()
}

func (h *HAProxyGenerator) generateName(id string) string {
	if strings.Contains(id, "/") {
		parts := strings.Split(id, "/")

		// Remove empty part in case of leading '/' in id
		if parts[0] == "" {
			parts = parts[1:]
		}

		return strings.Join(parts, "_")
	}

	return id
}

func (h *HAProxyGenerator) sanitizeOptions(o string) []string {
	sanitizedOptions := []string{}

	options := strings.Split(o, "\n")
	for _, option := range options {
		option = strings.Trim(option, " ")
		if option == "" {
			continue
		}

		sanitizedOptions = append(sanitizedOptions, option)
	}

	return sanitizedOptions
}

func (h *HAProxyGenerator) readOptions(domain string, port int) (string, error) {
	optionPath := fmt.Sprintf("%s/%s_%d", h.c.OptionsPath, domain, port)

	_, err := os.Stat(optionPath)
	if err != nil {
		return "", nil
	} else {
		f, err := os.Open(optionPath)
		if err != nil {
			return "", err
		}

		defer f.Close()

		options, err := ioutil.ReadAll(f)
		if err != nil {
			return "", err
		}

		return string(options), nil
	}
}

func readExistingFile(fp string) (string, error) {
	if _, err := os.Stat(fp); err != nil {
		return "", errors.New(fmt.Sprintf("'%s' does not exist.", fp))
	}

	f, err := os.Open(fp)
	if err != nil {
		return "", err
	}

	defer f.Close()

	content, err := ioutil.ReadAll(f)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

func readConfig() *Config {
	var httpPort int

	binPath := os.Getenv("HAPROXY_BINARY_PATH")
	configFilePath := os.Getenv("HAPROXY_CONFIG_FILE_PATH")
	httpPortEnv := os.Getenv("HAPROXY_HTTP_PORT")
	optionsPath := os.Getenv("HAPROXY_OPTIONS_PATH")
	processPidPath := os.Getenv("HAPROXY_PID_PATH")

	if httpPortEnv == "" {
		httpPort = 0
	} else {
		httpPort, _ = strconv.Atoi(httpPortEnv)
	}

	return &Config{
		BinPath:        binPath,
		ConfigFilePath: configFilePath,
		HttpPort:       httpPort,
		OptionsPath:    optionsPath,
		ProcessPidPath: processPidPath,
	}
}

// Create a new HAProxyGenerator, reading configuration from environment variables.
func NewGenerator() *HAProxyGenerator {
	config := readConfig()

	return &HAProxyGenerator{
		c: config,
	}
}
