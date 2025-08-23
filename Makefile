.PHONY: all build clean spine spine-cli spine-ws run-ws

# 默认目标
all: build

# 创建 bin 目录
bin:
	mkdir -p bin

# 构建所有可执行文件
build: bin spine spine-cli spine-ws

# 构建 spine 服务器
spine: bin
	go build -o bin/spine ./cmd/spine/

# 构建 spine-cli 客户端
spine-cli: bin
	go build -o bin/spine-cli ./cmd/spine-cli/

# 构建 spine-ws WebSocket 服务器
spine-ws: bin
	go build -o bin/spine-ws ./cmd/spine-ws/

# 运行 WebSocket 服务器
run-ws: spine-ws
	./bin/spine-ws

# 运行主服务器
run: spine
	./bin/spine

# 运行命令行客户端
run-cli: spine-cli
	./bin/spine-cli

# 清理构建产物
clean:
	rm -rf bin/
