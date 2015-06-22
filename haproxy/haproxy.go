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
	BinaryPath     string `envconfig:"binary_path"`
	ConfigFilePath string `envconfig:"config_file_path"`
	Enabled        bool
	PidPath        string `envconfig:"pid_path"`
	TemplatePath   string `envconfig:"template_path"`
}

type HAProxyGenerator struct {
	c *Config
}

// Creates a new HAproxy config file and reloads HAProxy
func (h *HAProxyGenerator) Generate(services []*types.Service) {
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

func (h *HAProxyGenerator) config(services []*types.Service) string {
	globalConfig, err := readExistingFile(h.c.TemplatePath)
	if err != nil {
		log.ErrorLog.Error("Unable to read global config. Stopping HAProxy config generator: %s", err)
		return ""
	}

	var out bytes.Buffer

	tpl, err := tpl.New().New("haproxy").Parse(globalConfig)
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

	envconfig.Process("proxym_haproxy", &c)

	if c.Enabled {
		cg := NewGenerator(&c)

		manager.AddConfigGenerator(cg)
	}
}
