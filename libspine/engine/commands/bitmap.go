package commands

import (
	"fmt"
	"strconv"

	"spine-go/libspine/engine"
)

// RegisterBitmapCommands registers all bitmap commands
func RegisterBitmapCommands(registry *engine.CommandRegistry) error {
	commands := []engine.CommandHandler{
		&BitCountCommand{},
		&BitFieldCommand{},
		&BitFieldROCommand{},
		&BitOpCommand{},
		&BitPosCommand{},
		&GetBitCommand{},
		&SetBitCommand{},
	}

	for _, cmd := range commands {
		if err := registry.Register(cmd); err != nil {
			return fmt.Errorf("failed to register command %s: %w", cmd.GetInfo().Name, err)
		}
	}

	return nil
}

// SetBitCommand implements the SETBIT command
type SetBitCommand struct{}

func (c *SetBitCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs != 3 {
		return fmt.Errorf("wrong number of arguments for 'setbit' command")
	}

	keyValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	key, ok := keyValue.AsString()
	if !ok {
		return fmt.Errorf("invalid key")
	}

	offsetValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	offsetStr, ok := offsetValue.AsString()
	if !ok {
		return fmt.Errorf("invalid offset")
	}
	offset, err := strconv.ParseInt(offsetStr, 10, 64)
	if err != nil {
		return fmt.Errorf("bit offset is not an integer or out of range")
	}

	valueValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	valueStr, ok := valueValue.AsString()
	if !ok {
		return fmt.Errorf("invalid value")
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return fmt.Errorf("bit is not an integer or out of range")
	}

	oldBit, err := ctx.Database.BitmapStorage.SetBit(key, offset, value)
	if err != nil {
		return err
	}

	return ctx.RespWriter.WriteInteger(int64(oldBit))
}

func (c *SetBitCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "SETBIT",
		Summary:      "Sets or clears the bit at offset in the string value stored at key",
		Syntax:       "SETBIT key offset value",
		Categories:   []engine.CommandCategory{engine.CategoryBitmap},
		MinArgs:      3,
		MaxArgs:      3,
		ModifiesData: true,
	}
}

func (c *SetBitCommand) ModifiesData() bool { return true }

// GetBitCommand implements the GETBIT command
type GetBitCommand struct{}

func (c *GetBitCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs != 2 {
		return fmt.Errorf("wrong number of arguments for 'getbit' command")
	}

	keyValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	key, ok := keyValue.AsString()
	if !ok {
		return fmt.Errorf("invalid key")
	}

	offsetValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	offsetStr, ok := offsetValue.AsString()
	if !ok {
		return fmt.Errorf("invalid offset")
	}
	offset, err := strconv.ParseInt(offsetStr, 10, 64)
	if err != nil {
		return fmt.Errorf("bit offset is not an integer or out of range")
	}

	bit, err := ctx.Database.BitmapStorage.GetBit(key, offset)
	if err != nil {
		return err
	}

	return ctx.RespWriter.WriteInteger(int64(bit))
}

func (c *GetBitCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "GETBIT",
		Summary:      "Returns the bit value at offset in the string value stored at key",
		Syntax:       "GETBIT key offset",
		Categories:   []engine.CommandCategory{engine.CategoryBitmap},
		MinArgs:      2,
		MaxArgs:      2,
		ModifiesData: false,
	}
}

func (c *GetBitCommand) ModifiesData() bool { return false }

// BitCountCommand implements the BITCOUNT command
type BitCountCommand struct{}

func (c *BitCountCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs < 1 || nargs > 3 {
		return fmt.Errorf("wrong number of arguments for 'bitcount' command")
	}

	keyValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	key, ok := keyValue.AsString()
	if !ok {
		return fmt.Errorf("invalid key")
	}

	start := int64(0)
	end := int64(-1)

	if nargs >= 2 {
		startValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return err
		}
		startStr, ok := startValue.AsString()
		if !ok {
			return fmt.Errorf("invalid start")
		}
		start, err = strconv.ParseInt(startStr, 10, 64)
		if err != nil {
			return fmt.Errorf("value is not an integer or out of range")
		}
	}

	if nargs == 3 {
		endValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return err
		}
		endStr, ok := endValue.AsString()
		if !ok {
			return fmt.Errorf("invalid end")
		}
		end, err = strconv.ParseInt(endStr, 10, 64)
		if err != nil {
			return fmt.Errorf("value is not an integer or out of range")
		}
	}

	count, err := ctx.Database.BitmapStorage.BitCount(key, start, end)
	if err != nil {
		return err
	}

	return ctx.RespWriter.WriteInteger(count)
}

func (c *BitCountCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "BITCOUNT",
		Summary:      "Count set bits in a string",
		Syntax:       "BITCOUNT key [start end]",
		Categories:   []engine.CommandCategory{engine.CategoryBitmap},
		MinArgs:      1,
		MaxArgs:      3,
		ModifiesData: false,
	}
}

func (c *BitCountCommand) ModifiesData() bool { return false }

// BitPosCommand implements the BITPOS command
type BitPosCommand struct{}

func (c *BitPosCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs < 2 || nargs > 4 {
		return fmt.Errorf("wrong number of arguments for 'bitpos' command")
	}

	keyValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	key, ok := keyValue.AsString()
	if !ok {
		return fmt.Errorf("invalid key")
	}

	bitValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	bitStr, ok := bitValue.AsString()
	if !ok {
		return fmt.Errorf("invalid bit")
	}
	bit, err := strconv.Atoi(bitStr)
	if err != nil {
		return fmt.Errorf("bit is not an integer or out of range")
	}

	start := int64(0)
	end := int64(-1)

	if nargs >= 3 {
		startValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return err
		}
		startStr, ok := startValue.AsString()
		if !ok {
			return fmt.Errorf("invalid start")
		}
		start, err = strconv.ParseInt(startStr, 10, 64)
		if err != nil {
			return fmt.Errorf("value is not an integer or out of range")
		}
	}

	if nargs == 4 {
		endValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return err
		}
		endStr, ok := endValue.AsString()
		if !ok {
			return fmt.Errorf("invalid end")
		}
		end, err = strconv.ParseInt(endStr, 10, 64)
		if err != nil {
			return fmt.Errorf("value is not an integer or out of range")
		}
	}

	pos, err := ctx.Database.BitmapStorage.BitPos(key, bit, start, end)
	if err != nil {
		return err
	}

	return ctx.RespWriter.WriteInteger(pos)
}

func (c *BitPosCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "BITPOS",
		Summary:      "Find first bit set or clear in a string",
		Syntax:       "BITPOS key bit [start] [end]",
		Categories:   []engine.CommandCategory{engine.CategoryBitmap},
		MinArgs:      2,
		MaxArgs:      4,
		ModifiesData: false,
	}
}

func (c *BitPosCommand) ModifiesData() bool { return false }

// BitOpCommand implements the BITOP command
type BitOpCommand struct{}

func (c *BitOpCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs < 3 {
		return fmt.Errorf("wrong number of arguments for 'bitop' command")
	}

	operationValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	operation, ok := operationValue.AsString()
	if !ok {
		return fmt.Errorf("invalid operation")
	}

	destKeyValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	destKey, ok := destKeyValue.AsString()
	if !ok {
		return fmt.Errorf("invalid destination key")
	}

	keys := make([]string, nargs-2)
	for i := 0; i < nargs-2; i++ {
		keyValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return err
		}
		key, ok := keyValue.AsString()
		if !ok {
			return fmt.Errorf("invalid key")
		}
		keys[i] = key
	}

	resultLen, err := ctx.Database.BitmapStorage.BitOp(operation, destKey, keys)
	if err != nil {
		return err
	}

	return ctx.RespWriter.WriteInteger(resultLen)
}

func (c *BitOpCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "BITOP",
		Summary:      "Perform bitwise operations between strings",
		Syntax:       "BITOP operation destkey key [key ...]",
		Categories:   []engine.CommandCategory{engine.CategoryBitmap},
		MinArgs:      3,
		MaxArgs:      -1,
		ModifiesData: true,
	}
}

func (c *BitOpCommand) ModifiesData() bool { return true }

// BitFieldCommand implements the BITFIELD command
type BitFieldCommand struct{}

func (c *BitFieldCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs < 1 {
		return fmt.Errorf("wrong number of arguments for 'bitfield' command")
	}

	keyValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	key, ok := keyValue.AsString()
	if !ok {
		return fmt.Errorf("invalid key")
	}

	// For now, just consume all arguments and return empty array
	// Full BITFIELD implementation would be quite complex
	for i := 1; i < nargs; i++ {
		_, err := ctx.ReqReader.NextValue()
		if err != nil {
			return err
		}
	}

	_ = key
	return ctx.RespWriter.WriteArray([]interface{}{})
}

func (c *BitFieldCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "BITFIELD",
		Summary:      "Perform arbitrary bitfield integer operations on strings",
		Syntax:       "BITFIELD key [GET type offset] [SET type offset value] [INCRBY type offset increment] [OVERFLOW WRAP|SAT|FAIL]",
		Categories:   []engine.CommandCategory{engine.CategoryBitmap},
		MinArgs:      1,
		MaxArgs:      -1,
		ModifiesData: true,
	}
}

func (c *BitFieldCommand) ModifiesData() bool { return true }

// BitFieldROCommand implements the BITFIELD_RO command
type BitFieldROCommand struct{}

func (c *BitFieldROCommand) Execute(ctx *engine.CommandContext) error {
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return err
	}

	if nargs < 1 {
		return fmt.Errorf("wrong number of arguments for 'bitfield_ro' command")
	}

	keyValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return err
	}
	key, ok := keyValue.AsString()
	if !ok {
		return fmt.Errorf("invalid key")
	}

	// For now, just consume all arguments and return empty array
	// Full BITFIELD_RO implementation would be quite complex
	for i := 1; i < nargs; i++ {
		_, err := ctx.ReqReader.NextValue()
		if err != nil {
			return err
		}
	}

	_ = key
	return ctx.RespWriter.WriteArray([]interface{}{})
}

func (c *BitFieldROCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "BITFIELD_RO",
		Summary:      "Perform arbitrary bitfield integer operations on strings (read-only)",
		Syntax:       "BITFIELD_RO key [GET type offset] [GET type offset ...]",
		Categories:   []engine.CommandCategory{engine.CategoryBitmap},
		MinArgs:      1,
		MaxArgs:      -1,
		ModifiesData: false,
	}
}

func (c *BitFieldROCommand) ModifiesData() bool { return false }
