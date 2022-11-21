package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"text/template"

	"github.com/JackalLabs/jackal-provider/jprov/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	tmos "github.com/tendermint/tendermint/libs/os"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/jackalLabs/canine-chain/app"
	tmcfg "github.com/tendermint/tendermint/config"
	tmlog "github.com/tendermint/tendermint/libs/log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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

func NewDefaultContext() *Context {
	return NewContext(
		viper.New(),
		DefaultConfig(),
		tmlog.NewTMLogger(tmlog.NewSyncWriter(os.Stdout)),
	)
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

func NewRootCmd() *cobra.Command {
	encodingConfig := app.MakeEncodingConfig()

	cfg := sdk.GetConfig()
	cfg.SetBech32PrefixForAccount(app.Bech32PrefixAccAddr, app.Bech32PrefixAccPub)
	cfg.SetBech32PrefixForValidator(app.Bech32PrefixValAddr, app.Bech32PrefixValPub)
	cfg.SetBech32PrefixForConsensusNode(app.Bech32PrefixConsAddr, app.Bech32PrefixConsPub)
	cfg.Seal()

	initClientCtx := client.Context{}.
		WithCodec(encodingConfig.Marshaler).
		WithInterfaceRegistry(encodingConfig.InterfaceRegistry).
		WithTxConfig(encodingConfig.TxConfig).
		WithLegacyAmino(encodingConfig.Amino).
		WithInput(os.Stdin).
		WithAccountRetriever(authtypes.AccountRetriever{}).
		WithBroadcastMode(flags.BroadcastBlock).
		WithHomeDir(types.DefaultAppHome).
		WithViper("")

	rootCmd := &cobra.Command{
		Use:   "jprovd",
		Short: "Provider Daemon (server)",
		Long:  "Jackal Lab's implimentation of a Jackal Protocol Storage Provider system.",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			// set the default command outputs
			cmd.SetOut(cmd.OutOrStdout())
			cmd.SetErr(cmd.ErrOrStderr())

			initClientCtx, err := client.ReadPersistentCommandFlags(initClientCtx, cmd.Flags())
			if err != nil {
				fmt.Println(err)
				return err
			}

			initClientCtx, err = ReadFromClientConfig(initClientCtx)
			if err != nil {
				fmt.Println(err)
				return err
			}

			if err := client.SetCmdClientContextHandler(initClientCtx, cmd); err != nil {
				fmt.Println(err)
				return err
			}

			return nil

			// return interceptConfigsPreRunHandler(cmd, "", nil)
		},
	}

	init := CmdInitProvider()
	AddTxFlagsToCmd(init)

	rootCmd.AddCommand(
		StartServerCommand(),
		ResetCommand(),
		init,
		DataCmd(),
		ClientCmd(),
	)

	return rootCmd
}
