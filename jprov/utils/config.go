package utils

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"text/template"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	tmflags "github.com/tendermint/tendermint/libs/cli/flags"
	tmos "github.com/tendermint/tendermint/libs/os"

	sdk "github.com/cosmos/cosmos-sdk/types"
	tmcfg "github.com/tendermint/tendermint/config"
	tmcli "github.com/tendermint/tendermint/libs/cli"
	tmlog "github.com/tendermint/tendermint/libs/log"
)

type Context struct {
	Viper  *viper.Viper
	Config *Config
	Logger tmlog.Logger
}

const ProviderContextKey = sdk.ContextKey("provider.context")

var configTemplate *template.Template

func NewContext(v *viper.Viper, config *Config, logger tmlog.Logger) *Context {
	return &Context{v, config, logger}
}

func DefaultBaseConfig() BaseConfig {
	return BaseConfig{
		LogLevel:  tmcfg.DefaultLogLevel,
		LogFormat: tmcfg.LogFormatPlain,
	}
}

// DefaultConfig returns a default configuration for a Tendermint node
func DefaultConfig() *Config {
	return &Config{
		BaseConfig: DefaultBaseConfig(),
	}
}

type BaseConfig struct {
	// chainID is unexposed and immutable but here for convenience
	//nolint:all
	chainID string

	// The root directory for all data.
	// This should be set in viper so it can unmarshal into this struct
	RootDir string `mapstructure:"home"`

	LogLevel string `mapstructure:"log_level"`

	// Output format: 'plain' (colored text) or 'json'
	LogFormat string `mapstructure:"log_format"`
}

type Config struct {
	BaseConfig `mapstructure:",squash"`
}

func (cfg BaseConfig) ValidateBasic() error {
	switch cfg.LogFormat {
	case tmcfg.LogFormatPlain, tmcfg.LogFormatJSON:
	default:
		return errors.New("unknown log_format (must be 'plain' or 'json')")
	}
	return nil
}

func (cfg *Config) ValidateBasic() error {
	if err := cfg.BaseConfig.ValidateBasic(); err != nil {
		return err
	}

	// if err := cfg.Instrumentation.ValidateBasic(); err != nil {
	// 	return fmt.Errorf("error in [instrumentation] section: %w", err)
	// }
	return nil
}

func WriteConfigFile(configFilePath string, config *Config) {
	var buffer bytes.Buffer

	if err := configTemplate.Execute(&buffer, config); err != nil {
		panic(err)
	}

	tmos.MustWriteFile(configFilePath, buffer.Bytes(), 0o644)
}

func (cfg *Config) SetRoot(root string) *Config {
	cfg.BaseConfig.RootDir = root
	return cfg
}

func NewDefaultContext() *Context {
	return NewContext(
		viper.New(),
		DefaultConfig(),
		tmlog.NewTMLogger(tmlog.NewSyncWriter(os.Stdout)),
	)
}

func interceptConfigs(rootViper *viper.Viper, customAppTemplate string, customConfig interface{}) (*Config, error) {
	rootDir := rootViper.GetString(flags.FlagHome)

	conf := DefaultConfig()

	conf.SetRoot(rootDir)

	return conf, nil
}

func GetServerContextFromCmd(cmd *cobra.Command) *Context {
	if v := cmd.Context().Value(ProviderContextKey); v != nil {
		serverCtxPtr := v.(*Context)
		return serverCtxPtr
	}

	return NewDefaultContext()
}

func SetCmdServerContext(cmd *cobra.Command, serverCtx *Context) error {
	v := cmd.Context().Value(ProviderContextKey)
	if v == nil {
		return errors.New("server context not set")
	}

	serverCtxPtr := v.(*Context)
	*serverCtxPtr = *serverCtx

	return nil
}

func InterceptConfigsPreRunHandler(cmd *cobra.Command, customAppConfigTemplate string, customAppConfig interface{}) error {
	serverCtx := NewDefaultContext()

	// Get the executable name and configure the viper instance so that environmental
	// variables are checked based off that name. The underscore character is used
	// as a separator
	executableName, err := os.Executable()
	if err != nil {
		return err
	}

	basename := path.Base(executableName)

	// Configure the viper instance
	if err := serverCtx.Viper.BindPFlags(cmd.Flags()); err != nil {
		return err
	}
	if err := serverCtx.Viper.BindPFlags(cmd.PersistentFlags()); err != nil {
		return err
	}
	serverCtx.Viper.SetEnvPrefix(basename)
	serverCtx.Viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	serverCtx.Viper.AutomaticEnv()

	// intercept configuration files, using both Viper instances separately
	config, err := interceptConfigs(serverCtx.Viper, customAppConfigTemplate, customAppConfig)
	if err != nil {
		return err
	}

	// return value is a tendermint configuration object
	serverCtx.Config = config
	if err = bindFlags(basename, cmd, serverCtx.Viper); err != nil {
		return err
	}
	logger := tmlog.NewTMLogger(tmlog.NewSyncWriter(os.Stdout))
	logger, err = tmflags.ParseLogLevel(config.LogLevel, logger, tmcfg.DefaultLogLevel)
	if err != nil {
		return err
	}

	// Check if the tendermint flag for trace logging is set
	// if it is then setup a tracing logger in this app as well
	if serverCtx.Viper.GetBool(tmcli.TraceFlag) {
		logger = tmlog.NewTracingLogger(logger)
	}

	serverCtx.Logger = logger

	return SetCmdServerContext(cmd, serverCtx)
}

func bindFlags(basename string, cmd *cobra.Command, v *viper.Viper) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("bindFlags failed: %v", r)
		}
	}()

	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		// Environment variables can't have dashes in them, so bind them to their equivalent
		// keys with underscores, e.g. --favorite-color to STING_FAVORITE_COLOR
		err = v.BindEnv(f.Name, fmt.Sprintf("%s_%s", basename, strings.ToUpper(strings.ReplaceAll(f.Name, "-", "_"))))
		if err != nil {
			panic(err)
		}

		err = v.BindPFlag(f.Name, f)
		if err != nil {
			panic(err)
		}

		// Apply the viper config value to the flag when the flag is not set and viper has a value
		if !f.Changed && v.IsSet(f.Name) {
			val := v.Get(f.Name)
			err = cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val))
			if err != nil {
				panic(err)
			}
		}
	})

	return err
}
