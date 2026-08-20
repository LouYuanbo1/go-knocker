package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/LouYuanbo1/go-knocker/config"
	"github.com/LouYuanbo1/go-knocker/knocker"
)

func main() {
	client := &http.Client{Timeout: 10 * time.Second}

	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = "config/knocker.json"
	}

	var k *knocker.Knocker
	var err error

	ext := filepath.Ext(cfgPath)
	if ext == ".yml" || ext == ".yaml" {
		k, err = config.LoadYAML(cfgPath, client)
	} else {
		k, err = config.Load(cfgPath, client)
	}
	if err != nil {
		log.Fatalf("load config failed: %v", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	go func() {
		<-sigChan
		log.Println("receive interrupt signal, stopping knocker...")
		k.Stop()

		time.AfterFunc(3*time.Second, func() {
			log.Println("stop timeout, force exit")
			os.Exit(1)
		})

		signal.Stop(sigChan)
	}()

	k.Run()
	log.Println("knocker exited")
}
