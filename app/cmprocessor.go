package watcher

import (
	"log"
	"os/exec"

	v1 "k8s.io/api/core/v1"
)

func ProcessAddUpdateCM(cm v1.ConfigMap, idx int) {
	cfg := GetWatcherConfig()
	log.Println("Add/Update File")
	cfgForLabeledResource := cfg.Resource.Labels[idx]
	url := cfgForLabeledResource.Request.URL
	method := cfgForLabeledResource.Request.Method
	if url != "" {
		log.Printf("making %s request to %s endpoint.\n", method, url)
	}
	script := cfgForLabeledResource.Script
	if script != "" {
		cmd := exec.Command(script)
		err := cmd.Start()
		if err != nil {
			log.Println(err)
		}
		err = cmd.Wait()
		if err != nil {
			log.Println(err)
		}
	}
	log.Println("Done processing add/update")
}

func ProcessDeleteCM(cm v1.ConfigMap, idx int) {
	cfg := GetWatcherConfig()
	log.Println("Delete File")
	cfgForLabeledResource := cfg.Resource.Labels[idx]
	url := cfgForLabeledResource.Request.URL
	method := cfgForLabeledResource.Request.Method
	if url != "" {
		log.Printf("making %s request to %s endpoint.\n", method, url)
	}
	script := cfgForLabeledResource.Script
	if script != "" {
		cmd := exec.Command(script)
		err := cmd.Start()
		if err != nil {
			log.Println(err)
		}
		err = cmd.Wait()
		if err != nil {
			log.Println(err)
		}
	}
	log.Println("Done processing delete")
}
