package haproxy

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/kelseyhightower/envconfig"
	"github.com/spf13/hugo/tpl"
	"github.com/wndhydrnt/proxym/log"
	"github.com/wndhydrnt/proxym/manager"
	"github.com/wndhydrnt/proxym/types"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

type Config struct {
	CheckCommand   string `envconfig:"check_command"`
	ConfigFilePath string `envconfig:"config_file_path"`
	Enabled        bool
	ReloadCommand  string `envconfig:"reload_command"`
	TemplatePath   string `envconfig:"template_path"`
}

type HAProxyGenerator struct {
	c *Config
}

// Creates a new HAproxy config file and reloads HAProxy
func (h *HAProxyGenerator) Generate(services []*types.Service) error {
	currentConfig, _ := readExistingFile(h.c.ConfigFilePath)
	newConfig := h.config(services)

	// No change. Do nothing.
	if currentConfig == newConfig {
		return nil
	}

	f, err := os.Create(h.c.ConfigFilePath)
	if err != nil {
		return errors.New(fmt.Sprintf("Unable to open config file for reading '%s': %s", h.c.ConfigFilePath, err))
	}
	defer f.Close()

	_, err = f.WriteString(newConfig)
	if err != nil {
		return errors.New(fmt.Sprintf("Unable to write config file '%s': %s", h.c.ConfigFilePath, err))
	}

	if h.c.CheckCommand != "" {
		cmd := exec.Command("/bin/bash", "-c", h.c.CheckCommand)

		var cmdErr bytes.Buffer
		cmd.Stderr = &cmdErr

		err := cmd.Run()
		if err != nil {
			return errors.New(fmt.Sprintf("Check of proxy configuration file failed: %s", cmdErr.String()))
		}
	}

	var reloadCommand string

	if strings.Contains(h.c.ReloadCommand, "%%s") {
		reloadCommand = fmt.Sprintf(h.c.ReloadCommand, h.c.ConfigFilePath)
	} else {
		reloadCommand = h.c.ReloadCommand
	}

	cmd := exec.Command("/bin/bash", "-c", reloadCommand)
	var cmdErr bytes.Buffer
	cmd.Stderr = &cmdErr

	log.AppLog.Info("Reloading proxy configuration")

	err = cmd.Run()
	if err != nil {
		return errors.New(fmt.Sprintf("Failed to reload proxy configuration -  Stderr of reload command: %s", cmdErr.String()))
	}
	return nil
}

func (h *HAProxyGenerator) config(services []*types.Service) string {
	globalConfig, err := readExistingFile(h.c.TemplatePath)
	if err != nil {
		log.ErrorLog.Error("Unable to read config template. Stopping proxy config generator: %s", err)
		return ""
	}

	var out bytes.Buffer

	tpl, err := tpl.New().New("proxy").Parse(globalConfig)
	if err != nil {
		log.ErrorLog.Error("%s", err)
		return ""
	}

	err = tpl.Execute(&out, services)
	if err != nil {
		log.ErrorLog.Error("%s", err)
		return ""
	}

	return removeEmptyLines(out.String()) + "\n"
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

// Helper that removes all lines in a string that consist only of spaces.
func removeEmptyLines(in string) string {
	newLines := []string{}

	lines := strings.Split(in, "\n")

	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			newLines = append(newLines, line)
		}
	}

	return strings.Join(newLines, "\n")
}

// Create a new HAProxyGenerator, reading configuration from environment variables.
func NewGenerator(c *Config) *HAProxyGenerator {
	return &HAProxyGenerator{
		c: c,
	}
}

func init() {
	var c Config

	envconfig.Process("proxym_proxy", &c)

	if c.Enabled {
		cg := NewGenerator(&c)

		manager.AddConfigGenerator(cg)
	}
}
