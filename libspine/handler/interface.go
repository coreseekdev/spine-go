package handler

import (
	"spine-go/libspine/transport"
)

// Handler 处理器接口
type Handler interface {
	Handle(ctx *transport.Context, req transport.Reader, res transport.Writer) error
}

// HandlerRegistry 处理器注册表
type HandlerRegistry struct {
	handlers map[string]Handler
}

// NewHandlerRegistry 创建新的处理器注册表
func NewHandlerRegistry() *HandlerRegistry {
	return &HandlerRegistry{
		handlers: make(map[string]Handler),
	}
}

// Register 注册处理器
func (r *HandlerRegistry) Register(path string, handler Handler) {
	r.handlers[path] = handler
}

// Get 获取处理器
func (r *HandlerRegistry) Get(path string) (Handler, bool) {
	handler, exists := r.handlers[path]
	return handler, exists
}

// HandlerFunc 处理器函数类型
type HandlerFunc func(ctx *transport.Context, req transport.Reader, res transport.Writer) error

// Handle 实现 Handler 接口
func (f HandlerFunc) Handle(ctx *transport.Context, req transport.Reader, res transport.Writer) error {
	return f(ctx, req, res)
}

// Middleware 中间件接口
type Middleware interface {
	Process(next Handler) Handler
}

// MiddlewareFunc 中间件函数类型
type MiddlewareFunc func(next Handler) Handler

// Process 实现 Middleware 接口
func (f MiddlewareFunc) Process(next Handler) Handler {
	return f(next)
}

// Chain 中间件链
type Chain struct {
	middlewares []Middleware
}

// NewChain 创建新的中间件链
func NewChain(middlewares ...Middleware) *Chain {
	return &Chain{
		middlewares: middlewares,
	}
}

// Then 应用中间件链到处理器
func (c *Chain) Then(handler Handler) Handler {
	result := handler
	for i := len(c.middlewares) - 1; i >= 0; i-- {
		result = c.middlewares[i].Process(result)
	}
	return result
}

// LoggerMiddleware 日志中间件
type LoggerMiddleware struct{}

func NewLoggerMiddleware() *LoggerMiddleware {
	return &LoggerMiddleware{}
}

func (m *LoggerMiddleware) Process(next Handler) Handler {
	return HandlerFunc(func(ctx *transport.Context, req transport.Reader, res transport.Writer) error {
		// 记录请求日志
		if ctx.ConnInfo != nil {
			// 这里可以添加日志逻辑
		}
		
		err := next.Handle(ctx, req, res)
		
		// 记录响应日志
		return err
	})
}

// AuthMiddleware 认证中间件
type AuthMiddleware struct {
	secret string
}

func NewAuthMiddleware(secret string) *AuthMiddleware {
	return &AuthMiddleware{secret: secret}
}

func (m *AuthMiddleware) Process(next Handler) Handler {
	return HandlerFunc(func(ctx *transport.Context, req transport.Reader, res transport.Writer) error {
		// 这里可以添加认证逻辑
		return next.Handle(ctx, req, res)
	})
}