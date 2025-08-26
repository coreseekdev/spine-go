package commands

import (
	"fmt"
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

	// Note: SELECT, PING, ECHO, QUIT commands are now in connection.go

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
