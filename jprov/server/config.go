package server

import (
	"log/slog"
	"path/filepath"

	"github.com/JackalLabs/jackal-provider/jprov/types"
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

	//RPC config
	chainID string
	// text or json
	output string
	// RPC node endpoint
	node string

	// proof
	postProofInterval int

	// littleHands; stray claimers
	threads int

	// listening port
	port int
}

type configFile struct {
	// DEBUG, INFO, WARN, ERROR
	LogLevel string
	// Output format: text or json
	LogFormat string

	//RPC config
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

		postProofInterval: configFile.PostProofInterval,
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
