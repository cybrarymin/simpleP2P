/*
Copyright © 2025 aminmoghaddam1377@gmail.com
*/
package cmd

import (
	"os"
	"time"

	observ "github.com/cybrarymin/simpleP2P/observability"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "simpleP2P",
	Short: "A simple data sharing p2p application",
	Long:  `A simple data sharing p2p application`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		main()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.simpleP2P.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.PersistentFlags().StringVar(&observ.CmdJaegerHostFlag, "jeager-host", "localhost", "Jaeger/jaeger-collector server address for sending opentelemetry traces")
	rootCmd.PersistentFlags().StringVar(&observ.CmdJaegerPortFlag, "jeager-port", "5317", "Jaeger/jaeger-collector server port for sending opentelemetry traces")
	rootCmd.PersistentFlags().DurationVar(&observ.CmdJaegerConnectionTimeout, "jeager-conn-timeout", time.Second*5, "connection will fail if it couldn't be established to jaeger host within this time")
	rootCmd.PersistentFlags().DurationVar(&observ.CmdSpanExportInterval, "jeager-trace-exporter-intervals", time.Second*5, "intervals which tracer batch exporter will send the traces to the jeager")
	rootCmd.PersistentFlags().StringVar(&CmdLogLevel, "log-level", "info", "application log level: debug, info, warn, error, fatal, panic, trace, disabled")
	rootCmd.PersistentFlags().StringVar(&CmdNodeAddr, "node-address", "0.0.0.0", "the address of the node or interface this node should listen and work on")
	rootCmd.PersistentFlags().StringVar(&CmdNodePort, "node-port", "6881", "the address of the node or interface this node should listen and work on")
	rootCmd.PersistentFlags().StringVar(&CmdBootstrapNodeAddr, "bootstrap-node-address", "localhost:6881", "the address of the bootstrap node in p2p network")
	rootCmd.PersistentFlags().BoolVar(&CmdBootstrapNode, "bootstrap-node", false, "specifies if this node is bootstrap or not")
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
