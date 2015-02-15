// A ConfigGenerator that generates a configuration file for HAProxy.
// Restarts HAProxy with the new config in case there are changes.
// It uses domain based routing for HTTP applications.
//
// It requires the following environment variables to be set:
// PROXYM_HAPROXY_BINARY_PATH - The absolute path to the binary of HAProxy
//
// PROXYM_HAPROXY_CONFIG_FILE_PATH - An absolute path where the generated config file will be stored.
//
// PROXYM_HAPROXY_ENABLED - Enable this module.
//
// PROXYM_HAPROXY_HTTP_PORT - Define the port under which applications available via HTTP will be reachable. This is usually 80.
//
// PROXYM_HAPROXY_OPTIONS_PATH - An absolute path to a file where 'global' options and 'default's of HAProxy are stored. The
//                        content of this file prepended to the rest of the config.
//
// PROXYM_HAPROXY_PID_PATH - An absolute where HAProxy stores its PID.
package haproxy

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/kelseyhightower/envconfig"
	"github.com/wndhydrnt/proxym/log"
	"github.com/wndhydrnt/proxym/manager"
	"github.com/wndhydrnt/proxym/types"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

type Config struct {
	BinaryPath     string `envconfig:"binary_path"`
	ConfigFilePath string `envconfig:"config_file_path"`
	Enabled        bool
	HttpPort       int    `envconfig:"http_port"`
	OptionsPath    string `envconfig:"options_path"`
	PidPath        string `envconfig:"pid_path"`
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

		log.ErrorLog.Error("Unable to open config file for reading '%s': %s", h.c.ConfigFilePath, err)
		return
	}

	defer f.Close()

	_, err = f.WriteString(newConfig)
	if err != nil {
		log.ErrorLog.Error("Unable to write config file '%s': %s", h.c.ConfigFilePath, err)
		return
	}

	cmdParts := []string{"-f", h.c.ConfigFilePath, "-p", h.c.PidPath}

	pid, err := readExistingFile(h.c.PidPath)
	if err == nil {
		cmdParts = append(cmdParts, "-sf")
		cmdParts = append(cmdParts, pid)
	}

	cmd := exec.Command(h.c.BinaryPath, cmdParts...)
	var cmdErr bytes.Buffer
	cmd.Stderr = &cmdErr

	log.AppLog.Info("Restarting HAProxy")

	err = cmd.Run()
	if err != nil {
		log.ErrorLog.Error("Failed to start HAProxy: %s", err)
		log.ErrorLog.Error("HAProxy Stderr: %s", cmdErr.String())
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
		log.ErrorLog.Error("Unable to read global config. Stopping HAProxy config generator: %s", err)
		return ""
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

	if len(services) == 0 {
		return ""
	}

	buffer.WriteString("\nfrontend http-in\n  bind *:80\n\n")

	for _, service := range services {
		name := h.generateName(service.Id)

		aclLine := fmt.Sprintf("  acl host_%s hdr(host) -i %s", name, service.Domain)
		aclLines = append(aclLines, aclLine)

		additionalDomains, err := h.readDomains(service.Domain, service.ServicePort)
		if err != nil {
			log.ErrorLog.Error("Error reading domains file: '%s'", err)
		}

		for _, d := range additionalDomains {
			aclLines = append(aclLines, fmt.Sprintf("  acl host_%s hdr(host) -i %s", name, d))
		}

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

		optionData, err := h.readConfig(service.Domain, service.ServicePort)
		if err != nil {
			log.ErrorLog.Error("Error reading options file: '%s'", err)
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

		optionData, err := h.readConfig(service.Domain, service.ServicePort)
		if err != nil {
			log.ErrorLog.Error("Error reading options file: '%s'", err)
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

func (h *HAProxyGenerator) readConfig(domain string, port int) (string, error) {
	optionPath := fmt.Sprintf("%s/%s_%d/config", h.c.OptionsPath, domain, port)

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

func (h *HAProxyGenerator) readDomains(domain string, port int) ([]string, error) {
	domains := []string{}

	domainsPath := fmt.Sprintf("%s/%s_%d/domains", h.c.OptionsPath, domain, port)

	_, err := os.Stat(domainsPath)
	if err != nil {
		return domains, nil
	} else {
		f, err := os.Open(domainsPath)
		if err != nil {
			return domains, err
		}

		defer f.Close()

		content, err := ioutil.ReadAll(f)
		if err != nil {
			return domains, err
		}

		sanitizedContent := strings.Trim(string(content), "\n")

		return strings.Split(sanitizedContent, "\n"), nil
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

// Create a new HAProxyGenerator, reading configuration from environment variables.
func NewGenerator(c *Config) *HAProxyGenerator {
	return &HAProxyGenerator{
		c: c,
	}
}

func init() {
	var c Config

	envconfig.Process("proxym_haproxy", &c)

	if c.Enabled {
		cg := NewGenerator(&c)

		manager.AddConfigGenerator(cg)
	}
}
