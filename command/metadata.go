package command

import (
	"time"

	"github.com/mackerelio/mackerel-agent/config"
	"github.com/mackerelio/mackerel-agent/metadata"
)

func metadataGenerators(conf *config.Config) []*metadata.Generator {
	generators := make([]*metadata.Generator, 0, len(conf.MetadataPlugins))

	for name, pluginConfig := range conf.MetadataPlugins {
		generator := &metadata.Generator{
			Name:   name,
			Config: pluginConfig,
		}
		logger.Debugf("Metadata plugin generator created: %#v %#v", generator, generator.Config)
		generators = append(generators, generator)
	}

	return generators
}

type metadataResult struct {
	namespace string
	metadata  interface{}
}

func runMetadataLoop(c *Context, termMetadataCh <-chan struct{}, quit <-chan struct{}) {
	resultCh := make(chan *metadataResult)
	for _, g := range c.Agent.MetadataGenerators {
		go runEachMetadataLoop(g, resultCh, quit)
	}

	exit := false
	for !exit {
		select {
		case <-time.After(1 * time.Minute):
		case <-termMetadataCh:
			logger.Debugf("received 'term' chan for metadata loop")
			exit = true
		}

		results := []*metadataResult{}
		hasResult := true
		for hasResult {
			select {
			case result := <-resultCh:
				results = append(results, result)
			default:
				hasResult = false
			}
		}

		for _, result := range results {
			err := c.API.PutMetadata(c.Host.ID, result.namespace, result.metadata)
			if err != nil {
				logger.Errorf("put metadata %q failed: %s", result.namespace, err.Error())
				continue
			}
		}
	}
}

func runEachMetadataLoop(g *metadata.Generator, resultCh chan<- *metadataResult, quit <-chan struct{}) {
	interval := g.Interval()
	nextInterval := 10 * time.Second
	nextTime := time.Now()

	for {
		select {
		case <-time.After(nextInterval):
			metadata, err := g.Fetch()

			// case for laptop sleep mode (now >> nextTime + interval)
			now := time.Now()
			nextInterval = interval - (now.Sub(nextTime) % interval)
			nextTime = now.Add(nextInterval)

			if err != nil {
				logger.Warningf("metadata %q: %s", g.Name, err.Error())
				continue
			}

			if !g.Differs(metadata) {
				logger.Debugf("skipping metadata %q: %v", g.Name, metadata)
				continue
			}
			_ = g.Save(metadata)

			logger.Debugf("generated metadata %q: %v", g.Name, metadata)
			resultCh <- &metadataResult{
				namespace: g.Name,
				metadata:  metadata,
			}

		case <-quit:
			return
		}
	}
}