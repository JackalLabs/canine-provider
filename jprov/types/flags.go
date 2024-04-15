package types

const (
	FlagThreads       = "threads"
	FlagInterval      = "interval"
	FlagMaxMisses     = "max-misses"
	FlagChunkSize     = "chunk-size"
	FlagStrayInterval = "stray-interval"
	FlagMessageSize   = "max-msg-size"
	FlagPort          = "port"
	FlagGasCap        = "gas-cap"
	FlagMaxFileSize   = "max-file-size"
	FlagQueueInterval = "queue-interval"
	FlagProviderName  = "moniker"
	FlagSleep         = "sleep"
	FlagDoReport      = "do-report"
)

const (
	DefaultThreads       = 3
	DefaultInterval      = 32
	DefaultMaxMisses     = 16
	DefaultChunkSize     = 10240
	DefaultStrayInterval = 20
	DefaultMessageSize   = 500000
	DefaultPort          = 3333
	DefaultGasCap        = 20000
	DefaultMaxFileSize   = 32000
	DefaultQueueInterval = 4
	DefaultSleep         = 250
	DefaultDoReport      = true
)
