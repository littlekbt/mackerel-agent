package command

import (
	"github.com/mackerelio/mackerel-agent/config"
	"github.com/mackerelio/mackerel-agent/metrics"
	metricsLinux "github.com/mackerelio/mackerel-agent/metrics/linux"
	"github.com/mackerelio/mackerel-agent/spec"
	specLinux "github.com/mackerelio/mackerel-agent/spec/linux"
)

func specGenerators() []spec.Generator {
	return []spec.Generator{
		&specLinux.KernelGenerator{},
		&specLinux.CPUGenerator{},
		&specLinux.MemoryGenerator{},
		&specLinux.BlockDeviceGenerator{},
		&specLinux.FilesystemGenerator{},
	}
}

func interfaceGenerator() spec.Generator {
	return &specLinux.InterfaceGenerator{}
}

func metricsGenerators(conf config.Config) []metrics.Generator {
	generators := []metrics.Generator{
		&metricsLinux.Loadavg5Generator{},
		&metricsLinux.CpuusageGenerator{Interval: 60},
		&metricsLinux.MemoryGenerator{},
		&metricsLinux.UptimeGenerator{},
		&metricsLinux.InterfaceGenerator{Interval: 60},
		&metricsLinux.DiskGenerator{Interval: 60},
	}
	for _, pluginConfig := range conf.Plugin["metrics"] {
		pluginGenerator := &metricsLinux.PluginGenerator{Config: pluginConfig}
		err := pluginGenerator.Init()

		if err == nil {
			generators = append(generators, pluginGenerator)
		} else {
			logger.Errorf("Error while loading plugin: %s", err)
		}
	}

	return generators
}
