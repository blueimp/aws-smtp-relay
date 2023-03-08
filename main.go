package main

import (
	"fmt"
	"os"

	receiver "github.com/blueimp/aws-smtp-relay/internal/receiver/aws_ses"
	"github.com/blueimp/aws-smtp-relay/internal/relay"
	"github.com/blueimp/aws-smtp-relay/internal/relay/config"
	"github.com/spf13/pflag"
)

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
	srv, err := relay.Server(cfg)
	if err == nil {
		err = srv.ListenAndServe()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
