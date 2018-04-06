package config

import (
	"errors"
	"time"

	"github.com/fsnotify/fsnotify"
)

type ConfigWatcher struct {
	NewConfig <-chan *Config
	Errors    <-chan error
	reload    chan<- bool
}

func Watch(path string) *ConfigWatcher {
	configChan := make(chan *Config, 1)
	errorChan := make(chan error, 1)
	reload := make(chan bool)

	go func() {
		defer close(configChan)
		defer close(errorChan)
		defer close(reload)

		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			errorChan <- err
			return
		}

		if err = watcher.Add(path); err != nil {
			errorChan <- err
			return
		}

		for {
			cfg, err := Load(path)
			if err != nil {
				errorChan <- err
				return
			}

			configChan <- cfg

			select {
			case evt := <-watcher.Events:
				if evt.Op == fsnotify.Remove || evt.Op == fsnotify.Rename {
					// The file was removed, which may be because it is being copied
					// over. We need to wait and see if it comes back.
					time.Sleep(100 * time.Millisecond)

					if err = watcher.Add(path); err != nil {
						errorChan <- errors.New("Config file was removed.")
						return
					}

				}

			case err = <-watcher.Errors:
				errorChan <- err
				return

			case <-reload:
				continue
			}

		}
	}()

	return &ConfigWatcher{
		NewConfig: configChan,
		Errors:    errorChan,
		reload:    reload,
	}
}

func (cw *ConfigWatcher) ForceReload() (*Config, error) {
	cw.reload <- true

	select {
	case cfg := <-cw.NewConfig:
		return cfg, nil

	case err := <-cw.Errors:
		return nil, err
	}
}
