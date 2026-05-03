// Package main is the entry point for the cmn-weave CLI client.
package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/ryo-arima/cmn-weave/pkg/client/controller"
	"github.com/ryo-arima/cmn-weave/pkg/client/repository"
	"github.com/ryo-arima/cmn-weave/pkg/client/share"
	"github.com/ryo-arima/cmn-weave/pkg/client/usecase"
	"github.com/ryo-arima/cmn-weave/pkg/config"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "client",
		Short: "cmn-weave CLI client",
		Long: `cmn-weave client submits work to the server over mTLS and reports results.

All commands require a valid client certificate issued from the operator CA.
Configure the certificate paths in the config file (default: etc/client.yaml).`,
		SilenceUsage: true,
	}

	var configFile string
	var outputFormat string
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "etc/client.yaml", "path to client config file")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "table", "output format: table, json, yaml")

	// Parse flags early so configFile is available before config.Load.
	_ = rootCmd.ParseFlags(os.Args[1:])

	var cfg config.YamlConfig
	if err := config.Load(configFile, &cfg); err != nil {
		log.Fatalf("load config %s: %v", configFile, err)
	}
	clientcfg := cfg.Application.Client
	tlsCfg, err := share.ClientTLSConfig(clientcfg.TLS)
	if err != nil {
		log.Fatalf("build tls config: %v", err)
	}

	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		controller.SetOutputFormat(outputFormat)
	}

	serverRepo := repository.NewHTTPServerClient(clientcfg.ServerEndpoint, tlsCfg)

	var adminRepo repository.AdminRepository
	if cfg.PostgreSQL.Host != "" {
		var err2 error
		adminRepo, err2 = repository.NewGORMAdminRepository(cfg.PostgreSQL.DSN())
		if err2 != nil {
			log.Fatalf("open db: %v", err2)
		}
	}
	uc := usecase.New(serverRepo, adminRepo)

	// bootstrap sub-commands (direct DB access, no server round-trip)
	bootstrapCmd := &cobra.Command{
		Use:   "bootstrap",
		Short: "Bootstrap system resources",
	}
	if cfg.PostgreSQL.Host != "" {
		bootstrapCmd.AddCommand(controller.NewBootstrapDBCmd(uc))
	} else {
		bootstrapCmd.AddCommand(&cobra.Command{
			Use:   "db",
			Short: "Bootstrap the database schema",
			RunE: func(cmd *cobra.Command, args []string) error {
				return fmt.Errorf("postgres_dsn is not configured in %s", configFile)
			},
		})
	}

	rootCmd.AddCommand(controller.NewDispatchCmd(uc))
	rootCmd.AddCommand(controller.NewStatusCmd(uc))
	rootCmd.AddCommand(controller.NewCancelCmd(uc))
	rootCmd.AddCommand(bootstrapCmd)

	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "print version",
		Run: func(cmd *cobra.Command, args []string) {
			b, _ := os.ReadFile("VERSION")
			fmt.Printf("cmn-weave client %s\n", strings.TrimSpace(string(b)))
		},
	})

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
