package commands

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"spine-go/libspine/engine"
	"spine-go/libspine/engine/storage/stream"
)

// XADD Command
type XAddCommand struct{}

func (c *XAddCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs < 3 {
		return fmt.Errorf("wrong number of arguments for 'xadd' command")
	}

	// Read key
	keyValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	key, ok := keyValue.AsString()
	if !ok {
		return fmt.Errorf("invalid key")
	}

	// Read ID
	idValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	idStr, ok := idValue.AsString()
	if !ok {
		return fmt.Errorf("invalid ID")
	}

	id, err := stream.ParseStreamID(idStr)
	if err != nil {
		return err
	}

	// Read field-value pairs
	remainingArgs := nargs - 2
	if remainingArgs%2 != 0 {
		return fmt.Errorf("wrong number of arguments for XADD")
	}

	fields := make(map[string]string)
	for i := 0; i < remainingArgs; i += 2 {
		fieldValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return err
		}
		field, ok := fieldValue.AsString()
		if !ok {
			return fmt.Errorf("invalid field")
		}

		valueValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return err
		}
		value, ok := valueValue.AsString()
		if !ok {
			return fmt.Errorf("invalid value")
		}

		fields[field] = value
	}

	// Get stream storage
	streamStorage := ctx.Database.StreamStorage
	if streamStorage == nil {
		return fmt.Errorf("stream storage not available")
	}

	// Add entry
	resultID, err := streamStorage.XAdd(key, id, fields, 0, false)
	if err != nil {
		return err
	}

	return ctx.RespWriter.WriteBulkString(resultID.String())
}

func (c *XAddCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "XADD",
		Summary:      "Appends a new entry to a stream",
		Syntax:       "XADD key ID field value [field value ...]",
		Categories:   []engine.CommandCategory{engine.CategoryStream},
		MinArgs:      4,
		MaxArgs:      -1,
		ModifiesData: true,
	}
}

func (c *XAddCommand) ModifiesData() bool { return true }

// XREAD Command
type XReadCommand struct{}

func (c *XReadCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs < 3 {
		return fmt.Errorf("wrong number of arguments for 'xread' command")
	}

	var count int64 = -1
	var timeout time.Duration = 0
	var streamsIndex int = 0

	// Parse optional arguments
	for i := 0; i < nargs; i++ {
		argValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return err
		}
		arg, ok := argValue.AsString()
		if !ok {
			return fmt.Errorf("invalid argument")
		}

		if strings.ToUpper(arg) == "COUNT" {
			i++
			if i >= nargs {
				return fmt.Errorf("syntax error")
			}
			countValue, err := ctx.ReqReader.NextValue()
			if err != nil {
				return err
			}
			countStr, ok := countValue.AsString()
			if !ok {
				return fmt.Errorf("invalid count")
			}
			count, err = strconv.ParseInt(countStr, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid count")
			}
		} else if strings.ToUpper(arg) == "BLOCK" {
			i++
			if i >= nargs {
				return fmt.Errorf("syntax error")
			}
			timeoutValue, err := ctx.ReqReader.NextValue()
			if err != nil {
				return err
			}
			timeoutStr, ok := timeoutValue.AsString()
			if !ok {
				return fmt.Errorf("invalid timeout")
			}
			timeoutMs, err := strconv.ParseInt(timeoutStr, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid timeout")
			}
			timeout = time.Duration(timeoutMs) * time.Millisecond
		} else if strings.ToUpper(arg) == "STREAMS" {
			streamsIndex = i + 1
			break
		} else {
			return fmt.Errorf("syntax error")
		}
	}

	if streamsIndex == 0 {
		return fmt.Errorf("syntax error")
	}

	// Read remaining arguments as streams and IDs
	remainingArgs := nargs - streamsIndex
	if remainingArgs%2 != 0 {
		return fmt.Errorf("unbalanced XREAD list of streams")
	}

	streamCount := remainingArgs / 2
	streams := make([]string, streamCount)
	ids := make([]stream.StreamID, streamCount)

	// Read stream names
	for i := 0; i < streamCount; i++ {
		streamValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return err
		}
		streamName, ok := streamValue.AsString()
		if !ok {
			return fmt.Errorf("invalid stream name")
		}
		streams[i] = streamName
	}

	// Read IDs
	for i := 0; i < streamCount; i++ {
		idValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return err
		}
		idStr, ok := idValue.AsString()
		if !ok {
			return fmt.Errorf("invalid ID")
		}
		id, err := stream.ParseStreamID(idStr)
		if err != nil {
			return err
		}
		ids[i] = id
	}

	// Get stream storage
	streamStorage := ctx.Database.StreamStorage
	if streamStorage == nil {
		return fmt.Errorf("stream storage not available")
	}

	// Execute read
	result, err := streamStorage.XRead(context.Background(), ctx.ClientID, streams, ids, count, timeout)
	if err != nil {
		return err
	}

	// Format response
	if len(result.Streams) == 0 {
		return ctx.RespWriter.WriteNull()
	}

	output := stream.FormatReadResult(result)
	return ctx.RespWriter.WriteArray(output)
}

func (c *XReadCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:       "XREAD",
		Summary:    "Return never seen elements in multiple streams",
		Syntax:     "XREAD [COUNT count] [BLOCK milliseconds] STREAMS key [key ...] id [id ...]",
		Categories: []engine.CommandCategory{engine.CategoryStream},
		MinArgs:    3,
		MaxArgs:    -1,
	}
}

func (c *XReadCommand) ModifiesData() bool { return false }

// Register stream commands
func RegisterStreamCommands(registry *engine.CommandRegistry) error {
	commands := []engine.CommandHandler{
		&XAddCommand{},
		&XReadCommand{},
	}

	for _, cmd := range commands {
		if err := registry.Register(cmd); err != nil {
			return fmt.Errorf("failed to register command %s: %w", cmd.GetInfo().Name, err)
		}
	}

	return nil
}
