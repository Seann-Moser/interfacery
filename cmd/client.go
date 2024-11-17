/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/Seann-Moser/go-serve/pkg/ctxLogger"
	"github.com/Seann-Moser/interfacery/pkg/parser"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"strings"

	"github.com/spf13/cobra"
)

// clientCmd represents the client command
var clientCmd = &cobra.Command{
	Use:   "client",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: ClientRunner,
}

func init() {
	clientCmd.Flags().AddFlagSet(Flags())
	rootCmd.AddCommand(clientCmd)
}

func Flags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("client", pflag.ExitOnError)
	fs.String("src-dir", "./", "")
	fs.String("dest-dir", "./pkg/client", "")
	fs.String("interface", "", "")
	return fs
}

func ClientRunner(cmd *cobra.Command, args []string) error {
	gofiles, err := parser.FindGoFilesWithInterfaces(viper.GetString("src-dir"), viper.GetString("interface"), strings.TrimPrefix(viper.GetString("dest-dir"), "./"))
	if err != nil {
		return err
	}
	for _, gofile := range gofiles {
		ctxLogger.Info(cmd.Context(), "Generating client for "+gofile.FilePath+"", zap.Strings("interfaces", gofile.Interfaces))

		err = parser.GenerateHTTPHandlers(cmd.Context(), gofile, gofile.PackageName, viper.GetString("dest-dir"), "")
		if err != nil {
			return err

		}

	}
	return nil
}
