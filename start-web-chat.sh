#!/bin/bash

echo "正在启动 Spine Chat Web 服务器..."

# 创建 bin 目录如果不存在
mkdir -p bin

# 构建 WebSocket 服务器
echo "构建中..."
go build -o bin/spine-ws ./cmd/spine-ws/

if [ $? -eq 0 ]; then
    echo "构建成功！"
    echo "正在启动服务器..."
    echo "服务器将在 http://localhost:8081 启动"
    echo "WebSocket 连接: ws://localhost:8081/ws"
    echo "按 Ctrl+C 停止服务器"
    echo ""
    
    # 启动服务器
    ./bin/spine-ws
else
    echo "构建失败！"
    exit 1
fi