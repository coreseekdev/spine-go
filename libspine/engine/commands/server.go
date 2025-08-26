package commands

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"time"

	"spine-go/libspine/engine"
)

// RegisterServerCommands registers all server-related commands
func RegisterServerCommands(registry *engine.CommandRegistry) error {
	commands := []engine.CommandHandler{
		&InfoCommand{},
		&ConfigGetCommand{},
		&ConfigSetCommand{},
		&FlushAllCommand{},
		&TimeCommand{},
		&CommandCommand{},
		&SaveCommand{},
		&LastSaveCommand{},
	}

	for _, cmd := range commands {
		if err := registry.Register(cmd); err != nil {
			return fmt.Errorf("failed to register command %s: %w", cmd.GetInfo().Name, err)
		}
	}

	return nil
}

// InfoCommand implements the INFO command
type InfoCommand struct{}

func (c *InfoCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	var section string
	if nargs == 1 {
		// Get specific section
		sectionValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return err
		}
		section, _ = sectionValue.AsString()
	}

	info := c.buildInfoResponse(section)
	return ctx.RespWriter.WriteBulkString(info)
}

func (c *InfoCommand) buildInfoResponse(section string) string {
	var info strings.Builder

	if section == "" || section == "server" {
		info.WriteString("# Server\r\n")
		info.WriteString("redis_version:7.0.0\r\n")
		info.WriteString("redis_git_sha1:00000000\r\n")
		info.WriteString("redis_git_dirty:0\r\n")
		info.WriteString("redis_build_id:spine-go\r\n")
		info.WriteString("redis_mode:standalone\r\n")
		info.WriteString("os:Linux\r\n")
		info.WriteString(fmt.Sprintf("arch_bits:%d\r\n", strconv.IntSize))
		info.WriteString("multiplexing_api:epoll\r\n")
		info.WriteString(fmt.Sprintf("atomicvar_api:atomic-builtin\r\n"))
		info.WriteString(fmt.Sprintf("gcc_version:%s\r\n", runtime.Version()))
		info.WriteString(fmt.Sprintf("process_id:%d\r\n", 1))
		info.WriteString(fmt.Sprintf("run_id:spine-go-instance\r\n"))
		info.WriteString(fmt.Sprintf("tcp_port:6379\r\n"))
		info.WriteString(fmt.Sprintf("uptime_in_seconds:%d\r\n", time.Now().Unix()))
		info.WriteString(fmt.Sprintf("uptime_in_days:%d\r\n", 0))
		info.WriteString(fmt.Sprintf("hz:10\r\n"))
		info.WriteString(fmt.Sprintf("configured_hz:10\r\n"))
		info.WriteString(fmt.Sprintf("lru_clock:%d\r\n", time.Now().Unix()))
		info.WriteString("executable:/usr/local/bin/spine-go\r\n")
		info.WriteString("config_file:\r\n")
		info.WriteString("\r\n")
	}

	if section == "" || section == "memory" {
		info.WriteString("# Memory\r\n")
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		info.WriteString(fmt.Sprintf("used_memory:%d\r\n", m.Alloc))
		info.WriteString(fmt.Sprintf("used_memory_human:%s\r\n", formatBytes(m.Alloc)))
		info.WriteString(fmt.Sprintf("used_memory_rss:%d\r\n", m.Sys))
		info.WriteString(fmt.Sprintf("used_memory_rss_human:%s\r\n", formatBytes(m.Sys)))
		info.WriteString(fmt.Sprintf("used_memory_peak:%d\r\n", m.TotalAlloc))
		info.WriteString(fmt.Sprintf("used_memory_peak_human:%s\r\n", formatBytes(m.TotalAlloc)))
		info.WriteString("mem_fragmentation_ratio:1.00\r\n")
		info.WriteString("mem_allocator:go\r\n")
		info.WriteString("\r\n")
	}

	if section == "" || section == "stats" {
		info.WriteString("# Stats\r\n")
		info.WriteString("total_connections_received:1\r\n")
		info.WriteString("total_commands_processed:0\r\n")
		info.WriteString("instantaneous_ops_per_sec:0\r\n")
		info.WriteString("total_net_input_bytes:0\r\n")
		info.WriteString("total_net_output_bytes:0\r\n")
		info.WriteString("instantaneous_input_kbps:0.00\r\n")
		info.WriteString("instantaneous_output_kbps:0.00\r\n")
		info.WriteString("rejected_connections:0\r\n")
		info.WriteString("sync_full:0\r\n")
		info.WriteString("sync_partial_ok:0\r\n")
		info.WriteString("sync_partial_err:0\r\n")
		info.WriteString("expired_keys:0\r\n")
		info.WriteString("evicted_keys:0\r\n")
		info.WriteString("keyspace_hits:0\r\n")
		info.WriteString("keyspace_misses:0\r\n")
		info.WriteString("pubsub_channels:0\r\n")
		info.WriteString("pubsub_patterns:0\r\n")
		info.WriteString("latest_fork_usec:0\r\n")
		info.WriteString("\r\n")
	}

	return strings.TrimSpace(info.String())
}

func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%dB", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func (c *InfoCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "INFO",
		Summary:      "Get information and statistics about the server",
		Syntax:       "INFO [section]",
		Categories:   []engine.CommandCategory{engine.CategoryServer},
		MinArgs:      0,
		MaxArgs:      1,
		ModifiesData: false,
	}
}

func (c *InfoCommand) ModifiesData() bool {
	return false
}

// ConfigGetCommand implements the CONFIG GET command
type ConfigGetCommand struct{}

func (c *ConfigGetCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs != 1 {
		return fmt.Errorf("wrong number of arguments for 'config get' command")
	}

	paramValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	parameter, ok := paramValue.AsString()
	if !ok {
		return fmt.Errorf("invalid parameter")
	}

	// Simple config values
	configs := map[string]string{
		"save":               "900 1 300 10 60 10000",
		"appendonly":         "no",
		"appendfsync":        "everysec",
		"maxmemory":          "0",
		"maxmemory-policy":   "noeviction",
		"timeout":            "0",
		"tcp-keepalive":      "300",
		"databases":          "16",
		"port":               "6379",
		"bind":               "127.0.0.1",
		"dir":                "/var/lib/redis",
		"dbfilename":         "dump.rdb",
		"requirepass":        "",
		"masterauth":         "",
		"slave-read-only":    "yes",
		"repl-diskless-sync": "no",
	}

	result := []interface{}{}
	if parameter == "*" {
		// Return all configs
		for key, value := range configs {
			result = append(result, key, value)
		}
	} else {
		// Return specific config
		if value, exists := configs[parameter]; exists {
			result = append(result, parameter, value)
		}
	}

	return ctx.RespWriter.WriteArray(result)
}

func (c *ConfigGetCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "CONFIG",
		Summary:      "Get configuration parameters",
		Syntax:       "CONFIG GET parameter",
		Categories:   []engine.CommandCategory{engine.CategoryServer},
		MinArgs:      2,
		MaxArgs:      2,
		ModifiesData: false,
	}
}

func (c *ConfigGetCommand) ModifiesData() bool {
	return false
}

// ConfigSetCommand implements the CONFIG SET command
type ConfigSetCommand struct{}

func (c *ConfigSetCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs != 2 {
		return fmt.Errorf("wrong number of arguments for 'config set' command")
	}

	paramValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	parameter, ok := paramValue.AsString()
	if !ok {
		return fmt.Errorf("invalid parameter")
	}

	valueParam, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	value, ok := valueParam.AsString()
	if !ok {
		return fmt.Errorf("invalid value")
	}

	// For now, just accept any config change
	_ = parameter
	_ = value

	return ctx.RespWriter.WriteSimpleString("OK")
}

func (c *ConfigSetCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "CONFIG",
		Summary:      "Set configuration parameters",
		Syntax:       "CONFIG SET parameter value",
		Categories:   []engine.CommandCategory{engine.CategoryServer},
		MinArgs:      3,
		MaxArgs:      3,
		ModifiesData: true,
	}
}

func (c *ConfigSetCommand) ModifiesData() bool {
	return true
}

// FlushAllCommand implements the FLUSHALL command
type FlushAllCommand struct{}

func (c *FlushAllCommand) Execute(ctx *engine.CommandContext) error {
	// For now, just clear current database
	if ctx.Database != nil {
		ctx.Database.FlushDB()
	}
	return ctx.RespWriter.WriteSimpleString("OK")
}

func (c *FlushAllCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "FLUSHALL",
		Summary:      "Remove all keys from all databases",
		Syntax:       "FLUSHALL [ASYNC]",
		Categories:   []engine.CommandCategory{engine.CategoryServer},
		MinArgs:      0,
		MaxArgs:      1,
		ModifiesData: true,
	}
}

func (c *FlushAllCommand) ModifiesData() bool {
	return true
}

// TimeCommand implements the TIME command
type TimeCommand struct{}

func (c *TimeCommand) Execute(ctx *engine.CommandContext) error {
	now := time.Now()
	seconds := now.Unix()
	microseconds := now.UnixMicro() % 1000000

	result := []interface{}{
		fmt.Sprintf("%d", seconds),
		fmt.Sprintf("%d", microseconds),
	}

	return ctx.RespWriter.WriteArray(result)
}

func (c *TimeCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "TIME",
		Summary:      "Return the current server time",
		Syntax:       "TIME",
		Categories:   []engine.CommandCategory{engine.CategoryServer},
		MinArgs:      0,
		MaxArgs:      0,
		ModifiesData: false,
	}
}

func (c *TimeCommand) ModifiesData() bool {
	return false
}

// CommandCommand implements the COMMAND command
type CommandCommand struct{}

func (c *CommandCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs == 0 {
		// Return all commands
		commands := ctx.Engine.GetCommandRegistry().ListCommands()
		result := []interface{}{}
		
		for name, info := range commands {
			cmdInfo := []interface{}{
				name,
				int64(info.MinArgs),
				[]interface{}{}, // flags
				int64(0),        // first key
				int64(0),        // last key
				int64(0),        // step
			}
			result = append(result, cmdInfo)
		}
		
		return ctx.RespWriter.WriteArray(result)
	}

	// Handle subcommands like COMMAND COUNT, COMMAND INFO, etc.
	subcommandValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	subcommand, ok := subcommandValue.AsString()
	if !ok {
		return fmt.Errorf("invalid subcommand")
	}

	switch strings.ToUpper(subcommand) {
	case "COUNT":
		commands := ctx.Engine.GetCommandRegistry().ListCommands()
		return ctx.RespWriter.WriteInteger(int64(len(commands)))
	case "INFO":
		// Return info for specific commands
		result := []interface{}{}
		for i := 1; i < nargs; i++ {
			cmdValue, err := ctx.ReqReader.NextValue()
			if err != nil {
				continue
			}
			cmdName, ok := cmdValue.AsString()
			if !ok {
				continue
			}
			
			commands := ctx.Engine.GetCommandRegistry().ListCommands()
			if info, exists := commands[strings.ToUpper(cmdName)]; exists {
				cmdInfo := []interface{}{
					info.Name,
					int64(info.MinArgs),
					[]interface{}{}, // flags
					int64(0),        // first key
					int64(0),        // last key
					int64(0),        // step
				}
				result = append(result, cmdInfo)
			} else {
				result = append(result, nil)
			}
		}
		return ctx.RespWriter.WriteArray(result)
	default:
		return fmt.Errorf("unknown subcommand: %s", subcommand)
	}
}

func (c *CommandCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "COMMAND",
		Summary:      "Get array of Redis command details",
		Syntax:       "COMMAND [subcommand [argument [argument ...]]]",
		Categories:   []engine.CommandCategory{engine.CategoryServer},
		MinArgs:      0,
		MaxArgs:      -1,
		ModifiesData: false,
	}
}

func (c *CommandCommand) ModifiesData() bool {
	return false
}

// SaveCommand implements the SAVE command
type SaveCommand struct{}

func (c *SaveCommand) Execute(ctx *engine.CommandContext) error {
	// For now, just return OK (no actual persistence implemented)
	return ctx.RespWriter.WriteSimpleString("OK")
}

func (c *SaveCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "SAVE",
		Summary:      "Synchronously save the dataset to disk",
		Syntax:       "SAVE",
		Categories:   []engine.CommandCategory{engine.CategoryServer},
		MinArgs:      0,
		MaxArgs:      0,
		ModifiesData: false,
	}
}

func (c *SaveCommand) ModifiesData() bool {
	return false
}

// LastSaveCommand implements the LASTSAVE command
type LastSaveCommand struct{}

func (c *LastSaveCommand) Execute(ctx *engine.CommandContext) error {
	// Return current time as last save time
	return ctx.RespWriter.WriteInteger(time.Now().Unix())
}

func (c *LastSaveCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "LASTSAVE",
		Summary:      "Get the UNIX time stamp of the last successful save to disk",
		Syntax:       "LASTSAVE",
		Categories:   []engine.CommandCategory{engine.CategoryServer},
		MinArgs:      0,
		MaxArgs:      0,
		ModifiesData: false,
	}
}

func (c *LastSaveCommand) ModifiesData() bool {
	return false
}
