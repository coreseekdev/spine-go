# Spine-Go E2E 测试框架

这是一个针对 spine-go 项目 chat 服务的端到端（E2E）测试框架，使用实际代码进行测试而非 Mock。

## 特性

- **真实环境测试**: 使用实际的服务器和客户端代码
- **多协议支持**: 支持 TCP、WebSocket、Unix Socket 协议
- **完整生命周期管理**: 在单个测试中完成服务启动、客户端连接、结果检查
- **消息验证**: 验证消息传递的正确性和广播功能
- **连接管理**: 验证连接状态和服务器连接数
- **跨协议测试**: 测试不同协议间的消息传递

## 架构组件

### 1. TestServerManager (`server_manager.go`)
- 管理测试服务器的生命周期
- 自动分配测试端口
- 支持多协议同时启动
- 提供服务器状态查询

### 2. TestClient (`test_client.go`)
- 提供统一的客户端接口
- 支持 TCP、WebSocket、Unix Socket 客户端
- 实现聊天协议的所有操作（连接、发送消息、加入/离开聊天等）

### 3. 验证器 (`test_validator.go`)
- **MessageValidator**: 验证消息内容和广播
- **ResponseValidator**: 验证服务器响应
- **ConnectionValidator**: 验证连接状态

### 4. E2E 测试套件 (`e2e_test.go`)
- 提供完整的测试用例
- 统一的测试环境管理
- 多种测试场景实现

## 使用方法

### 运行所有测试
```bash
cd test/e2e
go test -v
```

### 运行特定测试
```bash
# 测试 TCP 基本聊天功能
go test -v -run TestTCPBasicChat

# 测试 WebSocket 多客户端广播
go test -v -run TestWebSocketMultiClientBroadcast

# 测试跨协议通信
go test -v -run TestCrossProtocolCommunication
```

## 测试用例

### 1. 基本聊天测试
- 客户端连接服务器
- 加入聊天
- 发送消息
- 验证连接状态

### 2. 多客户端广播测试
- 多个客户端同时连接
- 验证消息广播到所有客户端
- 检查服务器连接数

### 3. 跨协议通信测试
- TCP 和 WebSocket 客户端同时连接
- 验证不同协议间的消息传递

### 4. 连接管理测试
- 测试连接建立和断开
- 验证服务器连接清理

## 扩展测试框架

### 添加新的测试客户端
1. 实现 `TestClient` 接口
2. 在 `TestClientFactory` 中添加创建逻辑

### 添加新的验证器
1. 创建验证器结构体
2. 实现相应的验证方法
3. 在测试套件中集成

### 添加新的测试用例
1. 在 `E2ETestSuite` 中添加测试方法
2. 创建对应的 `Test*` 函数
3. 使用现有的验证器进行断言

## 注意事项

1. **端口分配**: 测试框架自动分配可用端口，避免冲突
2. **资源清理**: 每个测试后自动清理服务器和客户端连接
3. **超时处理**: 所有网络操作都有超时保护
4. **错误处理**: 详细的错误信息帮助调试

## 依赖

- Go 1.21+
- github.com/gorilla/websocket
- spine-go/libspine (本地模块)

## 目录结构

```
test/e2e/
├── README.md              # 本文档
├── go.mod                 # Go 模块定义
├── server_manager.go      # 测试服务器管理
├── test_client.go         # 测试客户端实现
├── test_validator.go      # 验证器实现
└── e2e_test.go           # 测试用例
```
