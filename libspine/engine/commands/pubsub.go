package commands

import (
	"fmt"

	"spine-go/libspine/engine"
	"spine-go/libspine/transport"
)

// RegisterPubSubCommands 注册 pub/sub 命令
func RegisterPubSubCommands(registry *engine.CommandRegistry) error {
	// SUBSCRIBE 命令
	subscribeCmd := &SubscribeCommand{}
	if err := registry.Register(subscribeCmd); err != nil {
		return err
	}

	// UNSUBSCRIBE 命令
	unsubscribeCmd := &UnsubscribeCommand{}
	if err := registry.Register(unsubscribeCmd); err != nil {
		return err
	}

	// PSUBSCRIBE 命令
	psubscribeCmd := &PSubscribeCommand{}
	if err := registry.Register(psubscribeCmd); err != nil {
		return err
	}

	// PUNSUBSCRIBE 命令
	punsubscribeCmd := &PUnsubscribeCommand{}
	if err := registry.Register(punsubscribeCmd); err != nil {
		return err
	}

	// PUBLISH 命令
	publishCmd := &PublishCommand{}
	if err := registry.Register(publishCmd); err != nil {
		return err
	}

	// PUBSUB 命令
	pubsubCmd := &PubSubCommand{}
	if err := registry.Register(pubsubCmd); err != nil {
		return err
	}

	return nil
}

// SubscribeCommand implements the SUBSCRIBE command
type SubscribeCommand struct{}

func (c *SubscribeCommand) Execute(ctx *engine.CommandContext) error {
	// 获取参数数量
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid command format")
	}

	if nargs < 1 {
		return ctx.RespWriter.WriteError("ERR wrong number of arguments for 'subscribe' command")
	}

	// 获取 PubSub Manager（需要从 engine 中获取）
	pubsubManager := ctx.Engine.GetPubSubManager()
	if pubsubManager == nil {
		return ctx.RespWriter.WriteError("ERR pubsub not available")
	}

	// 获取连接信息
	connInfo := getConnInfoFromContext(ctx)
	if connInfo == nil {
		return ctx.RespWriter.WriteError("ERR connection info not available")
	}

	// 订阅所有指定的频道
	for i := 0; i < nargs; i++ {
		channelValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return ctx.RespWriter.WriteError("ERR invalid channel name")
		}

		channelName, ok := channelValue.AsString()
		if !ok {
			return ctx.RespWriter.WriteError("ERR invalid channel name")
		}

		// 订阅频道
		err = pubsubManager.Subscribe(connInfo, channelName)
		if err != nil {
			return ctx.RespWriter.WriteError(fmt.Sprintf("ERR %s", err.Error()))
		}

		// 发送订阅确认消息 (RESP3 Push)
		channelCount, patternCount := pubsubManager.GetSubscriptionCount(connInfo.ID)
		totalCount := channelCount + patternCount

		elements := []interface{}{
			"subscribe",
			channelName,
			int64(totalCount),
		}
		err = ctx.RespWriter.WritePush(elements)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *SubscribeCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "SUBSCRIBE",
		Summary:      "Subscribe to channels",
		Syntax:       "SUBSCRIBE channel [channel ...]",
		Categories:   []engine.CommandCategory{engine.CategoryConnection},
		MinArgs:      1,
		MaxArgs:      -1,
		ModifiesData: false,
	}
}

func (c *SubscribeCommand) ModifiesData() bool {
	return false
}

// UnsubscribeCommand implements the UNSUBSCRIBE command
type UnsubscribeCommand struct{}

func (c *UnsubscribeCommand) Execute(ctx *engine.CommandContext) error {
	// 获取参数数量
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid command format")
	}

	// 获取 PubSub Manager
	pubsubManager := ctx.Engine.GetPubSubManager()
	if pubsubManager == nil {
		return ctx.RespWriter.WriteError("ERR pubsub not available")
	}

	// 获取连接信息
	connInfo := getConnInfoFromContext(ctx)
	if connInfo == nil {
		return ctx.RespWriter.WriteError("ERR connection info not available")
	}

	if nargs == 0 {
		// 如果没有指定频道，取消订阅所有频道
		if connInfo.Metadata != nil {
			if subs, ok := connInfo.Metadata[transport.MetadataSubscriptions].([]string); ok {
				for _, channelName := range subs {
					err = pubsubManager.Unsubscribe(connInfo, channelName)
					if err != nil {
						return ctx.RespWriter.WriteError(fmt.Sprintf("ERR %s", err.Error()))
					}

					// 发送取消订阅确认消息
					channelCount, patternCount := pubsubManager.GetSubscriptionCount(connInfo.ID)
					totalCount := channelCount + patternCount

					elements := []interface{}{
						"unsubscribe",
						channelName,
						int64(totalCount),
					}
					err = ctx.RespWriter.WritePush(elements)
					if err != nil {
						return err
					}
				}
			}
		}
	} else {
		// 取消订阅指定的频道
		for i := 0; i < nargs; i++ {
			channelValue, err := ctx.ReqReader.NextValue()
			if err != nil {
				return ctx.RespWriter.WriteError("ERR invalid channel name")
			}

			channelName, ok := channelValue.AsString()
			if !ok {
				return ctx.RespWriter.WriteError("ERR invalid channel name")
			}

			// 取消订阅频道
			err = pubsubManager.Unsubscribe(connInfo, channelName)
			if err != nil {
				return ctx.RespWriter.WriteError(fmt.Sprintf("ERR %s", err.Error()))
			}

			// 发送取消订阅确认消息
			channelCount, patternCount := pubsubManager.GetSubscriptionCount(connInfo.ID)
			totalCount := channelCount + patternCount

			elements := []interface{}{
				"unsubscribe",
				channelName,
				int64(totalCount),
			}
			err = ctx.RespWriter.WritePush(elements)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *UnsubscribeCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "UNSUBSCRIBE",
		Summary:      "Unsubscribe from channels",
		Syntax:       "UNSUBSCRIBE [channel [channel ...]]",
		Categories:   []engine.CommandCategory{engine.CategoryConnection},
		MinArgs:      0,
		MaxArgs:      -1,
		ModifiesData: false,
	}
}

func (c *UnsubscribeCommand) ModifiesData() bool {
	return false
}

// PSubscribeCommand implements the PSUBSCRIBE command
type PSubscribeCommand struct{}

func (c *PSubscribeCommand) Execute(ctx *engine.CommandContext) error {
	// 获取参数数量
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid command format")
	}

	if nargs < 1 {
		return ctx.RespWriter.WriteError("ERR wrong number of arguments for 'psubscribe' command")
	}

	// 获取 PubSub Manager
	pubsubManager := ctx.Engine.GetPubSubManager()
	if pubsubManager == nil {
		return ctx.RespWriter.WriteError("ERR pubsub not available")
	}

	// 获取连接信息
	connInfo := getConnInfoFromContext(ctx)
	if connInfo == nil {
		return ctx.RespWriter.WriteError("ERR connection info not available")
	}

	// 订阅所有指定的模式
	for i := 0; i < nargs; i++ {
		patternValue, err := ctx.ReqReader.NextValue()
		if err != nil {
			return ctx.RespWriter.WriteError("ERR invalid pattern")
		}

		pattern, ok := patternValue.AsString()
		if !ok {
			return ctx.RespWriter.WriteError("ERR invalid pattern")
		}

		// 订阅模式
		err = pubsubManager.PSubscribe(connInfo, pattern)
		if err != nil {
			return ctx.RespWriter.WriteError(fmt.Sprintf("ERR %s", err.Error()))
		}

		// 发送订阅确认消息
		channelCount, patternCount := pubsubManager.GetSubscriptionCount(connInfo.ID)
		totalCount := channelCount + patternCount

		elements := []interface{}{
			"psubscribe",
			pattern,
			int64(totalCount),
		}
		err = ctx.RespWriter.WritePush(elements)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *PSubscribeCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "PSUBSCRIBE",
		Summary:      "Subscribe to patterns",
		Syntax:       "PSUBSCRIBE pattern [pattern ...]",
		Categories:   []engine.CommandCategory{engine.CategoryConnection},
		MinArgs:      1,
		MaxArgs:      -1,
		ModifiesData: false,
	}
}

func (c *PSubscribeCommand) ModifiesData() bool {
	return false
}

// PUnsubscribeCommand implements the PUNSUBSCRIBE command
type PUnsubscribeCommand struct{}

func (c *PUnsubscribeCommand) Execute(ctx *engine.CommandContext) error {
	// 获取参数数量
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid command format")
	}

	// 获取 PubSub Manager
	pubsubManager := ctx.Engine.GetPubSubManager()
	if pubsubManager == nil {
		return ctx.RespWriter.WriteError("ERR pubsub not available")
	}

	// 获取连接信息
	connInfo := getConnInfoFromContext(ctx)
	if connInfo == nil {
		return ctx.RespWriter.WriteError("ERR connection info not available")
	}

	if nargs == 0 {
		// 如果没有指定模式，取消订阅所有模式
		if connInfo.Metadata != nil {
			if patterns, ok := connInfo.Metadata[transport.MetadataPatternSubs].([]string); ok {
				for _, pattern := range patterns {
					err = pubsubManager.PUnsubscribe(connInfo, pattern)
					if err != nil {
						return ctx.RespWriter.WriteError(fmt.Sprintf("ERR %s", err.Error()))
					}

					// 发送取消订阅确认消息
					channelCount, patternCount := pubsubManager.GetSubscriptionCount(connInfo.ID)
					totalCount := channelCount + patternCount

					elements := []interface{}{
						"punsubscribe",
						pattern,
						int64(totalCount),
					}
					err = ctx.RespWriter.WritePush(elements)
					if err != nil {
						return err
					}
				}
			}
		}
	} else {
		// 取消订阅指定的模式
		for i := 0; i < nargs; i++ {
			patternValue, err := ctx.ReqReader.NextValue()
			if err != nil {
				return ctx.RespWriter.WriteError("ERR invalid pattern")
			}

			pattern, ok := patternValue.AsString()
			if !ok {
				return ctx.RespWriter.WriteError("ERR invalid pattern")
			}

			// 取消订阅模式
			err = pubsubManager.PUnsubscribe(connInfo, pattern)
			if err != nil {
				return ctx.RespWriter.WriteError(fmt.Sprintf("ERR %s", err.Error()))
			}

			// 发送取消订阅确认消息
			channelCount, patternCount := pubsubManager.GetSubscriptionCount(connInfo.ID)
			totalCount := channelCount + patternCount

			elements := []interface{}{
				"punsubscribe",
				pattern,
				int64(totalCount),
			}
			err = ctx.RespWriter.WritePush(elements)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *PUnsubscribeCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "PUNSUBSCRIBE",
		Summary:      "Unsubscribe from patterns",
		Syntax:       "PUNSUBSCRIBE [pattern [pattern ...]]",
		Categories:   []engine.CommandCategory{engine.CategoryConnection},
		MinArgs:      0,
		MaxArgs:      -1,
		ModifiesData: false,
	}
}

func (c *PUnsubscribeCommand) ModifiesData() bool {
	return false
}

// PublishCommand implements the PUBLISH command
type PublishCommand struct{}

func (c *PublishCommand) Execute(ctx *engine.CommandContext) error {
	// 获取参数数量
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid command format")
	}

	if nargs != 2 {
		return ctx.RespWriter.WriteError("ERR wrong number of arguments for 'publish' command")
	}

	// 读取频道名
	channelValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid channel name")
	}

	channelName, ok := channelValue.AsString()
	if !ok {
		return ctx.RespWriter.WriteError("ERR invalid channel name")
	}

	// 读取消息
	messageValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid message")
	}

	message, ok := messageValue.AsString()
	if !ok {
		return ctx.RespWriter.WriteError("ERR invalid message")
	}

	// 获取 PubSub Manager
	pubsubManager := ctx.Engine.GetPubSubManager()
	if pubsubManager == nil {
		return ctx.RespWriter.WriteError("ERR pubsub not available")
	}

	// 发布消息
	subscriberCount := pubsubManager.Publish(channelName, message)

	// 返回接收消息的订阅者数量
	return ctx.RespWriter.WriteInteger(int64(subscriberCount))
}

func (c *PublishCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "PUBLISH",
		Summary:      "Publish message to channel",
		Syntax:       "PUBLISH channel message",
		Categories:   []engine.CommandCategory{engine.CategoryConnection},
		MinArgs:      2,
		MaxArgs:      2,
		ModifiesData: false,
	}
}

func (c *PublishCommand) ModifiesData() bool {
	return false
}

// PubSubCommand implements the PUBSUB command
type PubSubCommand struct{}

func (c *PubSubCommand) Execute(ctx *engine.CommandContext) error {
	// 获取参数数量
	nargs, err := ctx.ReqReader.NArgs()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid command format")
	}

	if nargs < 1 {
		return ctx.RespWriter.WriteError("ERR wrong number of arguments for 'pubsub' command")
	}

	// 读取子命令
	subCmdValue, err := ctx.ReqReader.NextValue()
	if err != nil {
		return ctx.RespWriter.WriteError("ERR invalid subcommand")
	}

	subCmd, ok := subCmdValue.AsString()
	if !ok {
		return ctx.RespWriter.WriteError("ERR invalid subcommand")
	}

	// 获取 PubSub Manager
	pubsubManager := ctx.Engine.GetPubSubManager()
	if pubsubManager == nil {
		return ctx.RespWriter.WriteError("ERR pubsub not available")
	}

	switch subCmd {
	case "CHANNELS":
		// 返回所有活跃的频道
		if nargs == 1 {
			// 返回所有频道
			channels := pubsubManager.GetAllChannels()
			channelsInterface := make([]interface{}, len(channels))
			for i, ch := range channels {
				channelsInterface[i] = ch
			}
			return ctx.RespWriter.WriteArray(channelsInterface)
		} else {
			// 返回匹配模式的频道
			patternValue, err := ctx.ReqReader.NextValue()
			if err != nil {
				return ctx.RespWriter.WriteError("ERR invalid pattern")
			}
			_, ok := patternValue.AsString()
			if !ok {
				return ctx.RespWriter.WriteError("ERR invalid pattern")
			}
			
			// TODO: 实现模式匹配过滤
			channels := pubsubManager.GetAllChannels()
			channelsInterface := make([]interface{}, len(channels))
			for i, ch := range channels {
				channelsInterface[i] = ch
			}
			return ctx.RespWriter.WriteArray(channelsInterface)
		}

	case "NUMSUB":
		// 返回指定频道的订阅者数量
		result := make([]interface{}, 0)
		for i := 1; i < nargs; i++ {
			channelValue, err := ctx.ReqReader.NextValue()
			if err != nil {
				return ctx.RespWriter.WriteError("ERR invalid channel name")
			}
			channelName, ok := channelValue.AsString()
			if !ok {
				return ctx.RespWriter.WriteError("ERR invalid channel name")
			}
			
			count := pubsubManager.GetChannelSubscribers(channelName)
			result = append(result, channelName, int64(count))
		}
		return ctx.RespWriter.WriteArray(result)

	case "NUMPAT":
		// 返回模式订阅的数量
		patterns := pubsubManager.GetAllPatterns()
		return ctx.RespWriter.WriteInteger(int64(len(patterns)))

	default:
		return ctx.RespWriter.WriteError("ERR unknown PUBSUB subcommand")
	}
}

func (c *PubSubCommand) GetInfo() *engine.CommandInfo {
	return &engine.CommandInfo{
		Name:         "PUBSUB",
		Summary:      "Inspect pub/sub state",
		Syntax:       "PUBSUB subcommand [argument [argument ...]]",
		Categories:   []engine.CommandCategory{engine.CategoryConnection},
		MinArgs:      1,
		MaxArgs:      -1,
		ModifiesData: false,
	}
}

func (c *PubSubCommand) ModifiesData() bool {
	return false
}

// getConnInfoFromContext 从命令上下文中获取连接信息
func getConnInfoFromContext(ctx *engine.CommandContext) *transport.ConnInfo {
	if ctx.TransportCtx != nil {
		return ctx.TransportCtx.ConnInfo
	}
	return nil
}
