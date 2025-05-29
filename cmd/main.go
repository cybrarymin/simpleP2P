package cmd

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/cybrarymin/simpleP2P/pkgs/node"
	"github.com/rs/zerolog"
)

var (
	Version              string
	BuildTime            string
	CmdLogLevel          string
	CmdNodeAddr          string
	CmdNodePort          string
	CmdBootstrapNodeAddr string
	CmdBootstrapNode     bool
)

func main() {
	ctx := context.Background()
	var logger zerolog.Logger
	loglvl, err := zerolog.ParseLevel(CmdLogLevel)
	if err != nil {
		log.Panicln("couldn't parse the loglevel")
	}

	if CmdLogLevel == zerolog.LevelTraceValue {
		logger = zerolog.New(os.Stdout).With().Timestamp().Caller().Stack().Logger().Level(loglvl)
	} else {
		logger = zerolog.New(os.Stdout).With().Timestamp().Logger().Level(loglvl)
	}

	// otelShutdown, err := observ.SetupOTelSDK(ctx, observ.CmdJaegerHostFlag, observ.CmdJaegerPortFlag, observ.CmdJaegerConnectionTimeout, observ.CmdSpanExportInterval)
	// if err != nil {
	// 	logger.Error().Err(err).Msg("couldn't initialize the opemTelemetry")
	// 	return
	// }

	nNode := node.NewNode(&logger, CmdNodeAddr+":"+CmdNodePort, CmdBootstrapNodeAddr, CmdBootstrapNode)

	shutdownChan := make(chan error)
	go graceFulShutdown(ctx, &logger, shutdownChan, nNode.Stop)

	logger.Info().
		Str("node_id", nNode.ID.String()).
		Bool("boostrap_node", nNode.IsBootstrap).
		Msgf("starting the node on %s:%s....", CmdNodeAddr, CmdNodePort)
	err = nNode.Start(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("couldn't start the server")
		return
	}

	for {
		if err := <-shutdownChan; err == nil {
			return
		} else {
			logger.Error().Err(err).Msg("failed to shutdown gracefully")
		}
	}
}

func graceFulShutdown(ctx context.Context, logger *zerolog.Logger, shutdownErrors chan error, shutdownFuncs ...func(context.Context) error) {

	sigChannel := make(chan os.Signal, 1)
	signal.Notify(sigChannel, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	// block until receiving a signal
	sig := <-sigChannel
	logger.Info().Msgf("received singal %s from os", sig.String())

	logger.Info().Msg("start shutting down the server.....")
	for _, fn := range shutdownFuncs {
		err := fn(ctx)
		if err != nil {
			shutdownErrors <- err
		}
	}

	logger.Info().Msg("server shutdown completed")
	shutdownErrors <- nil
}
