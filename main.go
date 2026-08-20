package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
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
	if strings.HasSuffix(strings.ToLower(cfgPath), ".yaml") || strings.HasSuffix(strings.ToLower(cfgPath), ".yml") {
		k, err = config.LoadYAML(cfgPath, client)
	} else {
		k, err = config.Load(cfgPath, client)
	}
	if err != nil {
		log.Fatal(err)
	}

	// 优雅退出
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt)
		<-sig
		k.Stop()
	}()

	k.Run()
}
