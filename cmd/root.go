package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	cfgFile string
	debug   bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "edena",
	Short: "Edena is a toolkit for collecting and recording HTTP, SMTP and DNS interactions.",
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) {},
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(os.Stdout, `███████╗██████╗ ███████╗███╗   ██╗ █████╗
██╔════╝██╔══██╗██╔════╝████╗  ██║██╔══██╗
█████╗  ██║  ██║█████╗  ██╔██╗ ██║███████║
██╔══╝  ██║  ██║██╔══╝  ██║╚██╗██║██╔══██║
███████╗██████╔╝███████╗██║ ╚████║██║  ██║
╚══════╝╚═════╝ ╚══════╝╚═╝  ╚═══╝╚═╝  ╚═╝

`)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "sets minimum log level to debug")
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is $HOME/.edena.toml)")
	if err := rootCmd.MarkPersistentFlagFilename("config", "toml"); err != nil {
		panic(err)
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".edena" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("toml")
		viper.SetConfigName(".edena")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

func newLogger(debug, pretty bool) (*zap.Logger, error) {
	config := zap.Config{
		Level:       zap.NewAtomicLevelAt(zap.InfoLevel),
		Development: debug,
		Encoding:    "json",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "ts",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			FunctionKey:    zapcore.OmitKey,
			MessageKey:     "message",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.RFC3339TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
	}

	if debug {
		config.Level.SetLevel(zap.DebugLevel)
	} else {
		config = zap.NewProductionConfig()
	}

	if pretty {
		config.Encoding = "console"
		config.EncoderConfig = zapcore.EncoderConfig{
			TimeKey:       "T",
			LevelKey:      "L",
			NameKey:       "N",
			CallerKey:     "C",
			FunctionKey:   zapcore.OmitKey,
			MessageKey:    "M",
			StacktraceKey: "S",
			LineEnding:    zapcore.DefaultLineEnding,
			EncodeLevel:   zapcore.CapitalColorLevelEncoder,
			EncodeTime: func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
				if t.Location() == time.UTC {
					enc.AppendString(t.Format("2006-01-02T15:04:05Z07:00"))
				} else {
					enc.AppendString(t.Format("2006-01-02 15:04:05 Z07:00"))
				}
			},
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		}
	}

	return config.Build()
}
