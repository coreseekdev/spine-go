package commands

import (
	"fmt"
	"strconv"

	"spine-go/libspine/engine"
	"spine-go/libspine/transport"
)

// RegisterConnectionCommands registers all connection-related commands
func RegisterConnectionCommands(registry *engine.CommandRegistry) error {
	commands := []engine.CommandHandler{
		&PingCommand{},
		&EchoCommand{},
		&SelectCommand{},
		&QuitCommand{},
		&ClientIDCommand{},
		&ClientInfoCommand{},
		&ClientKillCommand{},
		&ClientListCommand{},
		&ClientSetNameCommand{},
		&ClientGetNameCommand{},
	}

	for _, cmd := range commands {
		if err := registry.Register(cmd); err != nil {
			return fmt.Errorf("failed to register command %s: %w", cmd.GetInfo().Name, err)
		}
	}

	return nil
}

// PingCommand implements the PING command
type PingCommand struct{}

func (c *PingCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs == 0 {
		// PING without message
		return ctx.RespWriter.WriteSimpleString("PONG")
	} else if nargs == 1 {
		// PING with message
		messageValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return err
		}
		message, ok := messageValue.AsString()
		if !ok {
			return fmt.Errorf("invalid message argument")
		}
		return ctx.RespWriter.WriteBulkString(message)
	}

	return fmt.Errorf("wrong number of arguments for 'ping' command")
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
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs != 1 {
		return fmt.Errorf("wrong number of arguments for 'echo' command")
	}

	messageValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	message, ok := messageValue.AsString()
	if !ok {
		return fmt.Errorf("invalid message argument")
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

// SelectCommand implements the SELECT command
type SelectCommand struct{}

func (c *SelectCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs != 1 {
		return fmt.Errorf("wrong number of arguments for 'select' command")
	}

	dbValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	dbIndexStr, ok := dbValue.AsString()
	if !ok {
		return fmt.Errorf("invalid DB index")
	}

	dbIndex, err := strconv.Atoi(dbIndexStr)
	if err != nil {
		return fmt.Errorf("invalid DB index")
	}

	if dbIndex < 0 || dbIndex > 15 {
		return fmt.Errorf("DB index is out of range")
	}

	// Store selected DB in transport context metadata
	if ctx.TransportCtx != nil && ctx.TransportCtx.ConnInfo != nil {
		if ctx.TransportCtx.ConnInfo.Metadata == nil {
			ctx.TransportCtx.ConnInfo.Metadata = make(map[string]interface{})
		}
		ctx.TransportCtx.ConnInfo.Metadata[transport.MetadataSelectedDB] = dbIndex
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

// QuitCommand implements the QUIT command
type QuitCommand struct{}

func (c *QuitCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs != 0 {
		return fmt.Errorf("wrong number of arguments for 'quit' command")
	}

	// Send OK response before closing
	err = ctx.RespWriter.WriteSimpleString("OK")
	if err != nil {
		return err
	}

	// Mark connection for closure (implementation depends on transport layer)
	if ctx.TransportCtx != nil && ctx.TransportCtx.ConnInfo != nil {
		if ctx.TransportCtx.ConnInfo.Metadata == nil {
			ctx.TransportCtx.ConnInfo.Metadata = make(map[string]interface{})
		}
		ctx.TransportCtx.ConnInfo.Metadata["quit_requested"] = true
	}

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

func (c *QuitCommand) ModifiesData() bool {
	return false
}

// ClientIDCommand implements the CLIENT ID command
type ClientIDCommand struct{}

func (c *ClientIDCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs != 0 {
		return fmt.Errorf("wrong number of arguments for 'client id' command")
	}

	// Generate a simple client ID based on connection info
	clientID := int64(1)
	if ctx.TransportCtx != nil && ctx.TransportCtx.ConnInfo != nil && ctx.TransportCtx.ConnInfo.Remote != nil {
		// Use connection address hash as simple ID
		clientID = int64(len(ctx.TransportCtx.ConnInfo.Remote.String()))
	}

	return ctx.RespWriter.WriteInteger(clientID)
}

func (c *ClientIDCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "CLIENT",
		Summary:      "Returns the client ID for the current connection",
		Syntax:       "CLIENT ID",
		Categories:   []engine.CommandCategory{engine.CategoryConnection},
		MinArgs:      0,
		MaxArgs:      0,
		ModifiesData: false,
	}
}

func (c *ClientIDCommand) ModifiesData() bool {
	return false
}

// ClientInfoCommand implements the CLIENT INFO command
type ClientInfoCommand struct{}

func (c *ClientInfoCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs != 0 {
		return fmt.Errorf("wrong number of arguments for 'client info' command")
	}

	// Build client info string
	info := "id=1 addr=127.0.0.1:0 fd=0 name= age=0 idle=0 flags=N db=0 sub=0 psub=0 multi=-1 qbuf=0 qbuf-free=0 obl=0 oll=0 omem=0 events=r cmd=client"
	
	if ctx.TransportCtx != nil && ctx.TransportCtx.ConnInfo != nil {
		if ctx.TransportCtx.ConnInfo.Metadata != nil {
			if db, ok := ctx.TransportCtx.ConnInfo.Metadata[transport.MetadataSelectedDB]; ok {
				addr := "127.0.0.1:0"
				if ctx.TransportCtx.ConnInfo.Remote != nil {
					addr = ctx.TransportCtx.ConnInfo.Remote.String()
				}
				info = fmt.Sprintf("id=1 addr=%s fd=0 name= age=0 idle=0 flags=N db=%v sub=0 psub=0 multi=-1 qbuf=0 qbuf-free=0 obl=0 oll=0 omem=0 events=r cmd=client", 
					addr, db)
			}
		}
	}

	return ctx.RespWriter.WriteBulkString(info)
}

func (c *ClientInfoCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "CLIENT",
		Summary:      "Returns information about the current client connection",
		Syntax:       "CLIENT INFO",
		Categories:   []engine.CommandCategory{engine.CategoryConnection},
		MinArgs:      0,
		MaxArgs:      0,
		ModifiesData: false,
	}
}

func (c *ClientInfoCommand) ModifiesData() bool {
	return false
}

// ClientKillCommand implements the CLIENT KILL command
type ClientKillCommand struct{}

func (c *ClientKillCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs < 1 {
		return fmt.Errorf("wrong number of arguments for 'client kill' command")
	}

	// For simplicity, just return that no clients were killed
	return ctx.RespWriter.WriteInteger(0)
}

func (c *ClientKillCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "CLIENT",
		Summary:      "Kill the connection of a client",
		Syntax:       "CLIENT KILL [ip:port] [ID client-id] [TYPE normal|master|slave|pubsub] [ADDR ip:port] [SKIPME yes/no]",
		Categories:   []engine.CommandCategory{engine.CategoryConnection},
		MinArgs:      1,
		MaxArgs:      -1,
		ModifiesData: false,
	}
}

func (c *ClientKillCommand) ModifiesData() bool {
	return false
}

// ClientListCommand implements the CLIENT LIST command
type ClientListCommand struct{}

func (c *ClientListCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs > 2 {
		return fmt.Errorf("wrong number of arguments for 'client list' command")
	}

	// Build simple client list with current connection
	clientList := "id=1 addr=127.0.0.1:0 fd=0 name= age=0 idle=0 flags=N db=0 sub=0 psub=0 multi=-1 qbuf=0 qbuf-free=0 obl=0 oll=0 omem=0 events=r cmd=client"
	
	if ctx.TransportCtx != nil && ctx.TransportCtx.ConnInfo != nil {
		if ctx.TransportCtx.ConnInfo.Metadata != nil {
			if db, ok := ctx.TransportCtx.ConnInfo.Metadata[transport.MetadataSelectedDB]; ok {
				addr := "127.0.0.1:0"
				if ctx.TransportCtx.ConnInfo.Remote != nil {
					addr = ctx.TransportCtx.ConnInfo.Remote.String()
				}
				clientList = fmt.Sprintf("id=1 addr=%s fd=0 name= age=0 idle=0 flags=N db=%v sub=0 psub=0 multi=-1 qbuf=0 qbuf-free=0 obl=0 oll=0 omem=0 events=r cmd=client", 
					addr, db)
			}
		}
	}

	return ctx.RespWriter.WriteBulkString(clientList)
}

func (c *ClientListCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "CLIENT",
		Summary:      "Get the list of client connections",
		Syntax:       "CLIENT LIST [TYPE normal|master|replica|pubsub]",
		Categories:   []engine.CommandCategory{engine.CategoryConnection},
		MinArgs:      0,
		MaxArgs:      2,
		ModifiesData: false,
	}
}

func (c *ClientListCommand) ModifiesData() bool {
	return false
}

// ClientSetNameCommand implements the CLIENT SETNAME command
type ClientSetNameCommand struct{}

func (c *ClientSetNameCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs != 1 {
		return fmt.Errorf("wrong number of arguments for 'client setname' command")
	}

	nameValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	name, ok := nameValue.AsString()
	if !ok {
		return fmt.Errorf("invalid client name")
	}

	// Store client name in transport context metadata
	if ctx.TransportCtx != nil && ctx.TransportCtx.ConnInfo != nil {
		if ctx.TransportCtx.ConnInfo.Metadata == nil {
			ctx.TransportCtx.ConnInfo.Metadata = make(map[string]interface{})
		}
		ctx.TransportCtx.ConnInfo.Metadata["client_name"] = name
	}

	return ctx.RespWriter.WriteSimpleString("OK")
}

func (c *ClientSetNameCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "CLIENT",
		Summary:      "Set the current connection name",
		Syntax:       "CLIENT SETNAME connection-name",
		Categories:   []engine.CommandCategory{engine.CategoryConnection},
		MinArgs:      1,
		MaxArgs:      1,
		ModifiesData: false,
	}
}

func (c *ClientSetNameCommand) ModifiesData() bool {
	return false
}

// ClientGetNameCommand implements the CLIENT GETNAME command
type ClientGetNameCommand struct{}

func (c *ClientGetNameCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs != 0 {
		return fmt.Errorf("wrong number of arguments for 'client getname' command")
	}

	// Get client name from transport context metadata
	if ctx.TransportCtx != nil && ctx.TransportCtx.ConnInfo != nil && ctx.TransportCtx.ConnInfo.Metadata != nil {
		if name, ok := ctx.TransportCtx.ConnInfo.Metadata["client_name"]; ok {
			if nameStr, ok := name.(string); ok {
				return ctx.RespWriter.WriteBulkString(nameStr)
			}
		}
	}

	return ctx.RespWriter.WriteNull()
}

func (c *ClientGetNameCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "CLIENT",
		Summary:      "Get the current connection name",
		Syntax:       "CLIENT GETNAME",
		Categories:   []engine.CommandCategory{engine.CategoryConnection},
		MinArgs:      0,
		MaxArgs:      0,
		ModifiesData: false,
	}
}

func (c *ClientGetNameCommand) ModifiesData() bool {
	return false
}

