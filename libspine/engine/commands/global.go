package commands

import (
	"fmt"
	"strconv"
	"strings"

	"spine-go/libspine/engine"
)

// RegisterGlobalCommands registers global Redis commands
func RegisterGlobalCommands(registry *engine.CommandRegistry) error {
	// HELLO command
	helloCmd := &HelloCommand{}
	if err := registry.Register(helloCmd); err != nil {
		return err
	}
	registry.RegisterAlias("HELLO", "HI") // 添加别名

	// SELECT command
	selectCmd := &SelectCommand{}
	if err := registry.Register(selectCmd); err != nil {
		return err
	}

	// PING command
	pingCmd := &PingCommand{}
	if err := registry.Register(pingCmd); err != nil {
		return err
	}

	// ECHO command
	echoCmd := &EchoCommand{}
	if err := registry.Register(echoCmd); err != nil {
		return err
	}

	// QUIT command
	quitCmd := &QuitCommand{}
	if err := registry.Register(quitCmd); err != nil {
		return err
	}
	registry.RegisterAlias("QUIT", "EXIT") // 添加别名

	// HELP 命令
	helpCmd := &HelpCommand{}
	registry.Register(helpCmd)

	return nil
}

// HelloCommand implements the HELLO command
type HelloCommand struct{}

func (c *HelloCommand) Execute(ctx *engine.CommandContext) error {
	// HELLO command response
	response := map[string]interface{}{
		"server":  "spine-redis",
		"version": "1.0.0",
		"proto":   3, // RESP3
		"id":      1,
		"mode":    "standalone",
		"role":    "master",
		"modules": []string{},
	}

	// Write RESP3 map response
	return ctx.RespWriter.WriteMap(response)
}

func (c *HelloCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "HELLO",
		Summary:      "Handshake with Redis",
		Syntax:       "HELLO [protover [AUTH username password] [SETNAME clientname]]",
		Categories:   []engine.CommandCategory{engine.CategoryConnection},
		MinArgs:      0,
		MaxArgs:      -1,
		ModifiesData: false,
	}
}

func (c *HelloCommand) ModifiesData() bool {
	return false
}

// SelectCommand implements the SELECT command
type SelectCommand struct{}

func (c *SelectCommand) Execute(ctx *engine.CommandContext) error {
	// Read the database index argument
	dbValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR wrong number of arguments for 'select' command")
	}

	dbIndexStr, ok := dbValue.AsString()
	if !ok {
		return ctx.RespWriter.WriteError("ERR invalid database index")
	}

	// Convert to integer
	dbNum, err := strconv.Atoi(dbIndexStr)
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid DB index")
	}

	// Validate database number (Redis typically supports 0-15)
	if dbNum < 0 || dbNum > 15 {
		return ctx.RespWriter.WriteError(fmt.Sprintf("ERR invalid DB index: %d", dbNum))
	}

	// Set the database in the context
	ctx.Database = ctx.Engine.GetDatabase(dbNum)
	if ctx.Database == nil {
		return ctx.RespWriter.WriteError(fmt.Sprintf("ERR invalid DB index: %d", dbNum))
	}

	return ctx.RespWriter.WriteSimpleString("OK")
}

func (c *SelectCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "SELECT",
		Summary:      "Change the selected database",
		Syntax:       "SELECT index",
		Categories:   []engine.CommandCategory{engine.CategoryConnection},
		MinArgs:      1,
		MaxArgs:      1,
		ModifiesData: false,
	}
}

func (c *SelectCommand) ModifiesData() bool {
	return false
}

// PingCommand implements the PING command
type PingCommand struct{}

func (c *PingCommand) Execute(ctx *engine.CommandContext) error {
	// Get the number of arguments
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid command format")
	}

	// PING command can have 0 or 1 argument
	if nargs == 0 {
		// No arguments, return PONG
		return ctx.RespWriter.WriteSimpleString("PONG")
	} else if nargs == 1 {
		// Get the next argument value
		messageValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return ctx.RespWriter.WriteError("ERR invalid argument")
		}

		// Read the message argument
		message, ok := messageValue.AsString()
		if !ok {
			return ctx.RespWriter.WriteError("ERR invalid argument")
		}

		return ctx.RespWriter.WriteBulkString(message)
	} else {
		// Too many arguments
		return ctx.RespWriter.WriteError("ERR wrong number of arguments for 'ping' command")
	}
}

func (c *PingCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "PING",
		Summary:      "Ping the server",
		Syntax:       "PING [message]",
		Categories:   []engine.CommandCategory{engine.CategoryConnection},
		MinArgs:      0,
		MaxArgs:      1,
		ModifiesData: false,
	}
}

func (c *PingCommand) ModifiesData() bool {
	return false
}

// EchoCommand implements the ECHO command
type EchoCommand struct{}

func (c *EchoCommand) Execute(ctx *engine.CommandContext) error {
	// Read the message argument
	messageValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR wrong number of arguments for 'echo' command")
	}

	message, ok := messageValue.AsString()
	if !ok {
		return ctx.RespWriter.WriteError("ERR invalid message")
	}

	// Check for additional arguments (should be none)
	_, err = ctx.ReqReader.NextValue()
	if err == nil {
		return ctx.RespWriter.WriteError("ERR wrong number of arguments for 'echo' command")
	}

	return ctx.RespWriter.WriteBulkString(message)
}

func (c *EchoCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "ECHO",
		Summary:      "Echo the given string",
		Syntax:       "ECHO message",
		Categories:   []engine.CommandCategory{engine.CategoryConnection},
		MinArgs:      1,
		MaxArgs:      1,
		ModifiesData: false,
	}
}

func (c *EchoCommand) ModifiesData() bool {
	return false
}

// QuitCommand implements the QUIT command
type QuitCommand struct{}

func (c *QuitCommand) Execute(ctx *engine.CommandContext) error {
	// Send OK response and close connection
	err := ctx.RespWriter.WriteSimpleString("OK")
	if err != nil {
		return err
	}
	// Note: Connection closing should be handled by the transport layer
	return nil
}

func (c *QuitCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "QUIT",
		Summary:      "Close the connection",
		Syntax:       "QUIT",
		Categories:   []engine.CommandCategory{engine.CategoryConnection},
		MinArgs:      0,
		MaxArgs:      0,
		ModifiesData: false,
	}
}

func (cmd *QuitCommand) ModifiesData() bool {
	return false
}

// HelpCommand HELP 命令实现
type HelpCommand struct{}

func (cmd *HelpCommand) Execute(ctx *engine.CommandContext) error {
	// Try to read an argument
	cmdValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		// 如果没有指定命令，显示所有命令的帮助
		err := ctx.RespWriter.WriteBulkString("Redis commands help:")
		if err != nil {
			return err
		}

		// 获取所有注册的命令
		commands := ctx.Engine.GetCommandRegistry().ListCommands()

		// 按类别组织命令
		categories := map[engine.CommandCategory][]string{
			engine.CategoryConnection: {},
			engine.CategoryGeneric:    {},
			engine.CategoryString:     {},
			engine.CategoryHash:       {},
			engine.CategoryList:       {},
			engine.CategorySet:        {},
			engine.CategoryServer:     {},
			engine.CategoryRead:       {},
			engine.CategoryWrite:      {},
		}
		for cmdName, info := range commands {
			for _, category := range info.Categories {
				categories[category] = append(categories[category], cmdName)
			}
		}

		// 显示每个类别的命令
		for category, cmdNames := range categories {
			err = ctx.RespWriter.WriteBulkString(fmt.Sprintf("\n%s:", category))
			if err != nil {
				return err
			}
			for _, cmdName := range cmdNames {
				if info, exists := commands[cmdName]; exists {
					err = ctx.RespWriter.WriteBulkString(fmt.Sprintf("  %s - %s", info.Name, info.Summary))
					if err != nil {
						return err
					}
				}
			}
		}

		return ctx.RespWriter.WriteBulkString("\nUse HELP <command> for detailed information about a specific command.")
	} else {
		// 显示特定命令的详细帮助
		cmdName, ok := cmdValue.AsString()
		if !ok {
			return ctx.RespWriter.WriteError("ERR invalid command name")
		}
		cmdName = strings.ToUpper(cmdName)
		commands := ctx.Engine.GetCommandRegistry().ListCommands()
		info, exists := commands[cmdName]
		if !exists {
			return ctx.RespWriter.WriteError(fmt.Sprintf("Unknown command: %s", cmdName))
		}
		err = ctx.RespWriter.WriteBulkString(fmt.Sprintf("Command: %s", info.Name))
		if err != nil {
			return err
		}
		err = ctx.RespWriter.WriteBulkString(fmt.Sprintf("Summary: %s", info.Summary))
		if err != nil {
			return err
		}
		err = ctx.RespWriter.WriteBulkString(fmt.Sprintf("Syntax: %s", info.Syntax))
		if err != nil {
			return err
		}

		return nil
	}
}

func (cmd *HelpCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "HELP",
		Summary:      "Get help about Redis commands.",
		Syntax:       "HELP [command]",
		Categories:   []engine.CommandCategory{engine.CategoryConnection},
		MinArgs:      0,
		MaxArgs:      1,
		ModifiesData: false,
	}
}

func (cmd *HelpCommand) ModifiesData() bool {
	return false
}
