/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"log"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/Seann-Moser/go-serve/pkg/ctxLogger"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:           "interfacery",
	SilenceErrors: false,
	SilenceUsage:  true,
	PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
		return viper.BindPFlags(cmd.Flags())
	},
}

func Execute() error {
	logger, err := ctxLogger.NewLoggerFromFlags()
	if err != nil {
		log.Println("could not instantiate logger", err)
		return err
	}

	return rootCmd.ExecuteContext(ctxLogger.ConfigureCtx(logger, context.Background()))
}

func init() {
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	rootCmd.Flags().AddFlagSet(ctxLogger.Flags())
	viper.AutomaticEnv()
}
