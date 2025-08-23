class SpineChat {
    constructor() {
        this.ws = null;
        this.username = '';
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 5;
        this.reconnectDelay = 1000;
        this.messageId = 0;
        this.hasJoined = false; // 标记是否已经加入聊天
        
        this.initializeElements();
        this.setupEventListeners();
        this.connect();
    }
    
    initializeElements() {
        // 聊天界面元素
        this.chatMessages = document.getElementById('chat-messages');
        this.usernameInput = document.getElementById('username-input');
        this.messageInput = document.getElementById('message-input');
        this.setUsernameBtn = document.getElementById('set-username-btn');
        this.sendBtn = document.getElementById('send-btn');
        
        // 状态元素
        this.statusIndicator = document.getElementById('status-indicator');
        this.statusText = document.getElementById('status-text');
        this.onlineCount = document.getElementById('online-count');
    }
    
    setupEventListeners() {
        // 用户名设置
        this.setUsernameBtn.addEventListener('click', () => this.setUsername());
        this.usernameInput.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') {
                this.setUsername();
            }
        });
        
        // 消息发送
        this.sendBtn.addEventListener('click', () => this.sendMessage());
        this.messageInput.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') {
                this.sendMessage();
            }
        });
        
        // WebSocket 连接状态变化
        window.addEventListener('online', () => {
            if (this.ws === null || this.ws.readyState === WebSocket.CLOSED) {
                this.connect();
            }
        });
        
        window.addEventListener('offline', () => {
            this.updateConnectionStatus('disconnected');
        });
    }
    
    connect() {
        try {
            // 检查是否已经有活动连接
            if (this.ws && (this.ws.readyState === WebSocket.CONNECTING || this.ws.readyState === WebSocket.OPEN)) {
                console.log('已经有活动连接，不创建新连接');
                return;
            }
            
            this.updateConnectionStatus('connecting');
            console.log('开始创建新连接...');
            
            // 获取当前页面的协议和主机
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const host = window.location.host;
            const wsUrl = `${protocol}//${host}/ws`;
            
            console.log('连接到:', wsUrl);
            this.ws = new WebSocket(wsUrl);
            
            this.ws.onopen = () => {
                console.log('连接成功打开!');
                this.updateConnectionStatus('connected');
                this.reconnectAttempts = 0;
                this.addSystemMessage('连接成功！您可以开始聊天了。');
                
                // 重置 hasJoined 标志，因为这是一个新的连接
                console.log('重置 hasJoined 标志，当前值:', this.hasJoined);
                this.hasJoined = false;
                console.log('重置后 hasJoined 值:', this.hasJoined);
                
                // 如果之前有用户名，重新加入聊天
                if (this.username) {
                    console.log('检测到现有用户名:', this.username, '自动重新加入聊天');
                    // 设置一个延时，确保 WebSocket 连接完全建立
                    setTimeout(() => {
                        this.joinChat();
                    }, 500);
                }
            };
            
            this.ws.onmessage = (event) => {
                console.log('收到 WebSocket 消息:', event);
                console.log('消息类型:', typeof event.data);
                try {
                    this.handleMessage(event.data);
                } catch (error) {
                    console.error('处理消息时出错:', error);
                }
            };
            
            this.ws.onclose = (event) => {
                console.log('连接关闭代码:', event.code);
                console.log('连接关闭原因:', event.reason);
                console.log('连接是否正常关闭:', event.wasClean);
                this.updateConnectionStatus('disconnected');
                
                // 重置 hasJoined 标志，以便在重新连接后可以再次加入聊天
                this.hasJoined = false;
                
                this.addSystemMessage(`连接断开 (代码: ${event.code}), 正在尝试重新连接...`);
                this.reconnect();
            };
            
            this.ws.onerror = (error) => {
                console.error('WebSocket 错误:', error);
                this.updateConnectionStatus('error');
                this.addSystemMessage('连接出现错误，请检查网络连接。');
            };
            
        } catch (error) {
            console.error('连接创建失败:', error);
            this.updateConnectionStatus('error');
            this.addSystemMessage('无法创建连接，请稍后再试。');
        }
    }
    
    reconnect() {
        if (this.reconnectAttempts < this.maxReconnectAttempts) {
            this.reconnectAttempts++;
            // 使用指数退避策略，但限制最大延迟为 10 秒
            const delay = Math.min(this.reconnectDelay * Math.pow(1.5, this.reconnectAttempts - 1), 10000);
            
            console.log(`计划重连: 尝试 ${this.reconnectAttempts}/${this.maxReconnectAttempts}, 延迟 ${delay}ms`);
            
            setTimeout(() => {
                this.addSystemMessage(`正在尝试重新连接 (${this.reconnectAttempts}/${this.maxReconnectAttempts})...`);
                console.log(`执行重连: 尝试 ${this.reconnectAttempts}/${this.maxReconnectAttempts}`);
                
                // 在重连前关闭现有连接（如果有）
                if (this.ws) {
                    try {
                        this.ws.close();
                    } catch (e) {
                        console.log('关闭现有连接时出错:', e);
                    }
                }
                
                this.connect();
            }, delay);
        } else {
            this.addSystemMessage('重新连接失败，请刷新页面重试。');
            this.updateConnectionStatus('error');
            console.log('达到最大重连次数，停止重连');
        }
    }
    
    updateConnectionStatus(status) {
        this.statusIndicator.className = `status-indicator status-${status}`;
        
        switch (status) {
            case 'connected':
                this.statusText.textContent = '已连接';
                break;
            case 'connecting':
                this.statusText.textContent = '连接中...';
                break;
            case 'disconnected':
                this.statusText.textContent = '未连接';
                break;
            case 'error':
                this.statusText.textContent = '连接错误';
                break;
        }
    }
    
    setUsername() {
        const username = this.usernameInput.value.trim();
        
        if (!username) {
            alert('请输入用户名');
            return;
        }
        
        if (username.length > 20) {
            alert('用户名不能超过20个字符');
            return;
        }
        
        this.username = username;
        this.usernameInput.disabled = true;
        this.setUsernameBtn.disabled = true;
        this.messageInput.disabled = false;
        this.sendBtn.disabled = false;
        this.messageInput.focus();
        
        this.addSystemMessage(`欢迎 ${username}！您可以开始聊天了。`);
        
        // 如果 WebSocket 已连接，发送加入聊天请求
        // 在这里重置 hasJoined 标志，因为用户可能更改了用户名
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            // 强制重置 hasJoined 标志，确保用户更改用户名后可以重新加入聊天
            this.hasJoined = false;
            this.joinChat();
        }
    }
    
    joinChat() {
        console.log('调用 joinChat 方法，当前 hasJoined 值:', this.hasJoined);
        
        // 如果已经加入过聊天，不再重复发送 JOIN 请求
        if (this.hasJoined) {
            console.log('已经加入过聊天，不再发送 JOIN 请求');
            return;
        }
        
        console.log('发送 JOIN 请求加入聊天');
        const joinRequest = {
            method: 'JOIN',
            path: '/chat',
            data: {}
        };
        
        // 在发送请求前先设置标志，防止重复发送
        this.hasJoined = true;
        console.log('设置 hasJoined = true');
        
        this.sendRequest(joinRequest);
    }
    
    sendMessage() {
        const message = this.messageInput.value.trim();
        
        if (!message) {
            return;
        }
        
        if (message.length > 500) {
            alert('消息不能超过500个字符');
            return;
        }
        
        const messageRequest = {
            method: 'POST',
            path: '/chat',
            data: {
                user: this.username,
                message: message
            }
        };
        
        this.sendRequest(messageRequest);
        this.messageInput.value = '';
        this.messageInput.focus();
    }
    
    sendRequest(request) {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            const messageId = ++this.messageId;
            const requestWithId = {
                ...request,
                id: messageId.toString()
            };
            
            this.ws.send(JSON.stringify(requestWithId));
        } else {
            this.addSystemMessage('连接断开，无法发送消息。');
        }
    }
    
    handleMessage(data) {
        try {
            console.log('收到消息类型:', typeof data, data instanceof ArrayBuffer ? 'ArrayBuffer' : '');
            console.log('原始消息数据:', data);
            
            // 数据可能是二进制格式，需要处理
            let jsonString;
            let response;
            
            try {
                if (data instanceof ArrayBuffer) {
                    // 如果是二进制数据，假设前4字节是长度
                    try {
                        const view = new DataView(data);
                        const length = view.getUint32(0);
                        const messageData = data.slice(4);
                        jsonString = new TextDecoder().decode(messageData);
                        console.log('解析二进制数据, 长度:', length, '解析后:', jsonString);
                    } catch (binaryError) {
                        console.error('解析二进制数据出错:', binaryError);
                        // 尝试直接解析整个 ArrayBuffer
                        jsonString = new TextDecoder().decode(data);
                        console.log('尝试直接解析 ArrayBuffer:', jsonString);
                    }
                } else {
                    jsonString = data;
                    console.log('收到文本数据:', jsonString);
                }
                
                console.log('准备解析 JSON:', jsonString);
                response = JSON.parse(jsonString);
                console.log('解析后的 JSON 对象:', response);
            } catch (parseError) {
                console.error('JSON 解析错误:', parseError);
                console.log('尝试解析的原始字符串:', jsonString);
                // 尝试处理可能的特殊情况
                try {
                    // 检查是否有额外的字符
                    if (jsonString && typeof jsonString === 'string') {
                        // 尝试清理字符串并重新解析
                        const cleanedString = jsonString.trim();
                        response = JSON.parse(cleanedString);
                        console.log('清理后成功解析 JSON:', response);
                    } else {
                        throw new Error('无法清理 JSON 字符串');
                    }
                } catch (retryError) {
                    console.error('重试解析仍然失败:', retryError);
                    this.addSystemMessage('收到无法解析的消息，请刷新页面重试');
                    return; // 退出函数，不处理这个消息
                }
            }
            
            // 处理不同类型的响应
            if (response.data && response.data.user && response.data.message) {
                // 聊天消息
                this.addChatMessage(response.data);
            } else if (response.data && Array.isArray(response.data)) {
                // 消息历史
                this.loadMessageHistory(response.data);
            } else if (response.status === 200) {
                // 操作成功响应
                if (response.data && response.data.message) {
                    this.addSystemMessage(response.data.message);
                }
            } else if (response.error) {
                // 错误响应
                this.addSystemMessage(`错误: ${response.error}`);
            }
            
        } catch (error) {
            console.error('消息处理错误:', error);
            console.error('原始数据:', data);
        }
    }
    
    addChatMessage(messageData) {
        const messageDiv = document.createElement('div');
        messageDiv.className = `message ${messageData.user === this.username ? 'sent' : 'received'}`;
        
        const time = new Date(messageData.timestamp).toLocaleTimeString('zh-CN');
        
        messageDiv.innerHTML = `
            <div class="message-info">
                <span class="message-username">${this.escapeHtml(messageData.user)}</span>
                <span class="message-time">${time}</span>
            </div>
            <div class="message-content">${this.escapeHtml(messageData.message)}</div>
        `;
        
        this.chatMessages.appendChild(messageDiv);
        this.scrollToBottom();
    }
    
    addSystemMessage(text) {
        const messageDiv = document.createElement('div');
        messageDiv.className = 'message system';
        messageDiv.innerHTML = `
            <div class="message-content">${this.escapeHtml(text)}</div>
        `;
        
        this.chatMessages.appendChild(messageDiv);
        this.scrollToBottom();
    }
    
    loadMessageHistory(messages) {
        // 清空现有消息（除了系统消息）
        const systemMessages = this.chatMessages.querySelectorAll('.system');
        this.chatMessages.innerHTML = '';
        systemMessages.forEach(msg => this.chatMessages.appendChild(msg));
        
        // 加载历史消息
        messages.forEach(message => {
            this.addChatMessage(message);
        });
    }
    
    scrollToBottom() {
        this.chatMessages.scrollTop = this.chatMessages.scrollHeight;
    }
    
    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
    
    // 更新在线用户数量（如果服务器提供此信息）
    updateOnlineCount(count) {
        this.onlineCount.textContent = count;
    }
}

// 页面加载完成后初始化聊天客户端
document.addEventListener('DOMContentLoaded', () => {
    new SpineChat();
});

// 处理页面卸载时的清理工作
window.addEventListener('beforeunload', () => {
    if (window.spineChat && window.spineChat.ws) {
        window.spineChat.ws.close();
    }
});
