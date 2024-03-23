package watcher

import (
	"flag"
	"fmt"
	"log"
	"os"

	"k8s.io/apimachinery/pkg/util/yaml"
)

var configFile = flag.String("watcher.config", "", "path to watcher config file")

var config *WatcherConfig

func InitConfig() error {
	if config == nil {
		if *configFile == "" {
			log.Fatal("--watcher.config cannot be empty")
		}
		file, err := os.ReadFile(*configFile)
		if err != nil {
			return err
		}
		err = yaml.Unmarshal(file, &config)
		if err != nil {
			log.Println("inside error")
			return err
		}
		return validate()
	}
	return nil
}

func GetWatcherConfig() WatcherConfig {
	return *config
}

func validate() error {
	if config.Resource.Type == "" {
		return fmt.Errorf("resoure type cannot be empty")
	}
	if config.Output.Folder == "" {
		return fmt.Errorf("folder value cannot be empty")
	}
	return nil
}

type WatcherConfig struct {
	Kubeconfig string         `yaml:"kubeconfig"`
	Namespace  string         `yaml:"namespace"`
	Resource   ResourceConfig `yaml:"resource"`
	Output     OutputConfig   `yaml:"output"`
}

type BasicAuthConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type RequestConfig struct {
	URL       string          `yaml:"url"`
	Method    string          `yaml:"method"`
	Payload   string          `yaml:"payload"`
	BasicAuth BasicAuthConfig `yaml:"basicAuth"`
}

type OutputConfig struct {
	Folder           string `yaml:"folder"`
	FolderAnnotation string `yaml:"folderAnnotation"`
}

type Labels struct {
	Name    string        `yaml:"name"`
	Value   string        `yaml:"value"`
	Script  string        `yaml:"script"`
	Request RequestConfig `yaml:"request"`
}
type ResourceConfig struct {
	Type   string   `yaml:"type"`
	Labels []Labels `yaml:"labels"`
}
