package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	receiver "github.com/blueimp/aws-smtp-relay/internal/receiver/aws_ses"
	"github.com/blueimp/aws-smtp-relay/internal/relay/config"
	"github.com/blueimp/aws-smtp-relay/internal/relay/server"
	"github.com/spf13/pflag"
)

func dumpCfg(cfg *config.Config, observeCfg *receiver.Config) {
	entry := struct {
		Time        time.Time
		Component   string
		Cfg         interface{}
		ObserverCfg interface{}
	}{
		Time:        time.Now().UTC(),
		Component:   "aws-smtp-relay",
		Cfg:         cfg,
		ObserverCfg: observeCfg,
	}
	b, _ := json.Marshal(entry)
	fmt.Println(string(b))
}

func main() {
	pflag.Parse()
	cfg, err := config.Configure()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	observeCfg, err := receiver.ConfigureObserver()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	dumpCfg(cfg, observeCfg)

	if observeCfg != nil {
		go func() {
			obs, err := receiver.NewAWSSESObserver(observeCfg)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			err = obs.InitSQS()
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			obs.Observe()
		}()
	}
	srv, err := server.Server(cfg)
	if err == nil {
		err = srv.ListenAndServe()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
