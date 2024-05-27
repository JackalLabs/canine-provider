package server

import (
	"log/slog"
	"path/filepath"

	"github.com/JackalLabs/jackal-provider/jprov/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	configName = "client"
	configType = "toml"

	// default configs
	defaultLogLevel  = slog.LevelInfo
	defaultLogFormat = "text"

	defaultChainID = "jackal-1"
	defaultOutput  = "text"
	defaultNode    = "tcp://localhost:26657"

	defaultThreads = 3
	defaultPort    = 3333

	defaultPostProofInterval = 32
)

type config struct {
	// DEBUG, INFO, WARN, ERROR
	logLevel slog.Level
	// Output format: text or json
	logFormat string

	// RPC config
	chainID string
	// text or json
	output string
	// RPC node endpoint
	node string

	// proof
	postProofInterval int64

	// littleHands; stray claimers
	threads int

	// listening port
	port int

	// The amount of intervals a provider can miss their proofs before removing a file
	maxMisses int
	// The size of a single file chunk.
	chunkSize int64
	// The interval in seconds to check for new strays
	strayInterval int64
	// The max size of all messages in bytes to submit to the chain at one time.
	messageSize int
	// The maximum gas to be used per message.
	gasCap int
	// The maximum size allowed to be sent to this provider in mbs. (only for monitoring services)
	maxFileSize int
	// The time, in seconds, between running a queue loop.
	queueInterval int64
	// The name to identify this provider in block explorers.
	providerName string
	// Should this provider report deals (uses gas).
	doReport bool
}

type configFile struct {
	// DEBUG, INFO, WARN, ERROR
	LogLevel string
	// Output format: text or json
	LogFormat string

	// RPC config
	ChainID string
	// text or json
	Output string
	// RPC node endpoint
	Node string

	// proof
	PostProofInterval int

	// littleHands; stray claimers
	Threads int

	// listening port
	Port int
}

func defaultConfig() config {
	return config{
		logLevel:  defaultLogLevel,
		logFormat: defaultLogFormat,

		chainID: defaultChainID,
		output:  defaultOutput,
		node:    defaultNode,
	}
}

func defaultConfigFile() configFile {
	return configFile{
		LogLevel:  defaultLogLevel.String(),
		LogFormat: defaultLogFormat,

		ChainID: defaultChainID,
		Output:  defaultOutput,
		Node:    defaultNode,

		PostProofInterval: defaultPostProofInterval,
		Threads:           defaultThreads,
		Port:              defaultPort,
	}
}

func ConfigureConfigurator(home string) error {
	if home == "" {
		home = types.DefaultAppHome
	}

	viper.SetConfigName(configName)
	viper.SetConfigType(configType)

	configFileName := configName + "." + configType
	configPath := filepath.Join(home, configFileName)
	viper.AddConfigPath(configPath)

	return nil
}

func ParseConfigFile() (config, error) {
	if err := viper.ReadInConfig(); err != nil {
		return config{}, err
	}

	configFile := defaultConfigFile()
	err := viper.Unmarshal(configFile)
	if err != nil {
		return config{}, err
	}

	logLevel, err := parseLogLevel(configFile.LogLevel)
	if err != nil {
		return config{}, err
	}

	conf := config{
		logLevel:  logLevel,
		logFormat: configFile.LogFormat,

		chainID: configFile.ChainID,
		output:  configFile.Output,
		node:    configFile.Node,

		postProofInterval: int64(configFile.PostProofInterval),
		threads:           configFile.Threads,
		port:              configFile.Port,
	}

	return conf, nil
}

func parseLogLevel(s string) (slog.Level, error) {
	var level slog.Level
	err := level.UnmarshalText([]byte(s))

	return level, err
}

func ParseCmdFlags(cmd *cobra.Command, config config) (config, error) {
	if cmd.Flags().Changed(types.FlagThreads) {
		threads, err := cmd.Flags().GetUint(types.FlagThreads)
		if err != nil {
			return config, err
		}
		config.threads = int(threads)
	}

	if cmd.Flags().Changed(types.FlagInterval) {
		interval, err := cmd.Flags().GetInt64(types.FlagInterval)
		if err != nil {
			return config, err
		}
		config.postProofInterval = interval
	}

	if cmd.Flags().Changed(types.FlagMaxMisses) {
		maxMisses, err := cmd.Flags().GetInt(types.FlagMaxMisses)
		if err != nil {
			return config, nil
		}
		config.maxMisses = maxMisses
	}

	if cmd.Flags().Changed(types.FlagChunkSize) {
		chunkSize, err := cmd.Flags().GetInt64(types.FlagChunkSize)
		if err != nil {
			return config, err
		}
		config.chunkSize = chunkSize
	}

	if cmd.Flags().Changed(types.FlagStrayInterval) {
		strayInterval, err := cmd.Flags().GetInt64(types.FlagStrayInterval)
		if err != nil {
			return config, err
		}
		config.strayInterval = strayInterval
	}

	if cmd.Flags().Changed(types.FlagMessageSize) {
		msgSize, err := cmd.Flags().GetInt(types.FlagMessageSize)
		if err != nil {
			return config, err
		}
		config.messageSize = msgSize
	}

	if cmd.Flags().Changed(types.FlagGasCap) {
		gasCap, err := cmd.Flags().GetInt(types.FlagGasCap)
		if err != nil {
			return config, err
		}
		config.gasCap = gasCap
	}

	if cmd.Flags().Changed(types.FlagMaxFileSize) {
		fileSize, err := cmd.Flags().GetInt(types.FlagMaxFileSize)
		if err != nil {
			return config, err
		}
		config.maxFileSize = fileSize
	}

	if cmd.Flags().Changed(types.FlagQueueInterval) {
		interval, err := cmd.Flags().GetInt64(types.FlagQueueInterval)
		if err != nil {
			return config, err
		}
		config.queueInterval = interval
	}

	if cmd.Flags().Changed(types.FlagProviderName) {
		name, err := cmd.Flags().GetString(types.FlagProviderName)
		if err != nil {
			return config, err
		}
		config.providerName = name
	}

	if cmd.Flags().Changed(types.FlagDoReport) {
		enable, err := cmd.Flags().GetBool(types.FlagDoReport)
		if err != nil {
			return config, err
		}
		config.doReport = enable
	}

	return config, nil
}
