        // WebSocket连接管理
        let ws = null;
        let isConnecting = false;
        let reconnectAttempts = 0;
        const maxReconnectAttempts = 5;
        
        // 模态框实例
        let serverConfigModal;
        
        document.addEventListener('DOMContentLoaded', function() {
            serverConfigModal = new bootstrap.Modal(document.getElementById('serverConfigModal'));
            
            // 认证方式切换
            document.querySelectorAll('input[name="auth_method"]').forEach(radio => {
                radio.addEventListener('change', function() {
                    if (this.value === 'password') {
                        document.getElementById('passwordAuth').style.display = 'block';
                        document.getElementById('keyAuth').style.display = 'none';
                    } else {
                        document.getElementById('passwordAuth').style.display = 'none';
                        document.getElementById('keyAuth').style.display = 'block';
                    }
                });
            });
            
            // 表单提交处理
            document.getElementById('serverConfigForm').addEventListener('submit', function(e) {
                e.preventDefault();
                saveServerConfig();
            });
            
            // 初始化WebSocket连接
            connectWebSocket();
            
            // 定时刷新服务器状态
            setInterval(refreshServerStatuses, 5000);
            
            // 加载保存的日志
            loadLogsFromStorage();
        });
        
        // 连接WebSocket
        function connectWebSocket() {
            if (isConnecting || (ws && ws.readyState === WebSocket.OPEN)) {
                return;
            }
            
            isConnecting = true;
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const wsUrl = `${protocol}//${window.location.host}/ws/progress`;
            
            try {
                ws = new WebSocket(wsUrl);
                
                ws.onopen = function() {
                    console.log('WebSocket 连接已建立');
                    isConnecting = false;
                    reconnectAttempts = 0;
                    showNotification('实时进度连接已建立', 'success', 2000);
                };
                
                ws.onmessage = function(event) {
                    try {
                        const data = JSON.parse(event.data);
                        handleProgressUpdate(data);
                    } catch (e) {
                        console.error('解析WebSocket消息失败:', e);
                    }
                };
                
                ws.onclose = function() {
                    console.log('WebSocket 连接已关闭');
                    isConnecting = false;
                    ws = null;
                    
                    // 尝试重连
                    if (reconnectAttempts < maxReconnectAttempts) {
                        reconnectAttempts++;
                        setTimeout(() => {
                            console.log(`尝试重连 WebSocket (${reconnectAttempts}/${maxReconnectAttempts})`);
                            connectWebSocket();
                        }, 2000 * reconnectAttempts);
                    }
                };
                
                ws.onerror = function(error) {
                    console.error('WebSocket 错误:', error);
                    isConnecting = false;
                };
            } catch (e) {
                console.error('创建WebSocket连接失败:', e);
                isConnecting = false;
            }
        }
        
        // 处理进度更新
        function handleProgressUpdate(data) {
            console.log('收到进度更新:', data);
            
            // 如果有服务器ID，更新对应服务器的状态
            if (data.server_id) {
                updateServerRowFromWebSocket(data.server_id, data);
            }
            
            // 添加到日志
            const logType = data.status === 'failed' ? 'error' : data.status === 'success' ? 'success' : 'info';
            addToLog(`[${data.server_name || '系统'}] ${data.message}`, logType);
        }
        
        // 从WebSocket数据更新服务器行
        function updateServerRowFromWebSocket(serverId, data) {
            const row = document.getElementById('server-row-' + serverId);
            if (!row) return;
            
            // 更新状态图标和文本
            const statusIcon = row.querySelector('.status-icon');
            const statusBadge = row.querySelector('.status-badge');
            const progressContainer = row.querySelector('.progress-container');
            
            if (statusIcon) {
                statusIcon.className = 'status-icon status-' + data.status;
            }
            
            if (statusBadge) {
                statusBadge.className = 'badge status-badge bg-' + 
                    (data.status === 'success' ? 'success' : 
                     data.status === 'failed' ? 'danger' : 
                     data.status === 'building' ? 'warning' : 
                     data.status === 'deploying' ? 'info' : 
                     data.status === 'paused' ? 'warning' : 'secondary');
                statusBadge.textContent = getStatusText(data.status);
            }
            
            // 更新进度条
            if (progressContainer && (data.status === 'building' || data.status === 'deploying')) {
                progressContainer.innerHTML = `
                    <div class="progress" style="height: 20px;">
                        <div class="progress-bar ${data.status === 'building' ? 'bg-warning' : 'bg-info'} progress-bar-striped progress-bar-animated" 
                             role="progressbar" style="width: ${data.progress}%" 
                             aria-valuenow="${data.progress}" aria-valuemin="0" aria-valuemax="100">
                            ${data.progress}%
                        </div>
                    </div>
                    ${data.speed ? `<div class="small text-muted mt-1"><i class="bi bi-speedometer2"></i> ${data.speed}</div>` : ''}
                `;
            } else if (progressContainer) {
                const iconClass = data.status === 'success' ? 'bi-check-circle-fill text-success' :
                                 data.status === 'failed' ? 'bi-x-circle-fill text-danger' : 
                                 'bi-dash-circle text-muted';
                progressContainer.innerHTML = `
                    <div class="text-center text-muted">
                        <i class="bi ${iconClass} fs-4"></i>
                    </div>
                `;
            }
        }
        
        // 通知系统
        function showNotification(message, type = 'info', duration = 4000) {
            const notification = document.createElement('div');
            notification.className = `notification ${type}`;
            notification.innerHTML = `
                <div style="display: flex; justify-content: space-between; align-items: center;">
                    <span>${message}</span>
                    <button onclick="this.parentElement.parentElement.remove()" 
                            style="background: none; border: none; color: white; font-size: 18px; cursor: pointer; margin-left: 10px;">×</button>
                </div>
            `;
            
            document.body.appendChild(notification);
            
            setTimeout(() => {
                notification.classList.add('show');
            }, 10);
            
            setTimeout(() => {
                if (notification.parentElement) {
                    notification.classList.remove('show');
                    setTimeout(() => {
                        if (notification.parentElement) {
                            notification.remove();
                        }
                    }, 300);
                }
            }, duration);
        }
        
        // 显示添加服务器模态框
        function showAddServerModal() {
            // 确保模态框已初始化
            if (!serverConfigModal) {
                serverConfigModal = new bootstrap.Modal(document.getElementById('serverConfigModal'));
            }
            
            document.getElementById('serverConfigForm').reset();
            document.getElementById('serverId').value = '';
            document.getElementById('serverPort').value = '22';
            document.getElementById('serverEnabled').checked = true;
            
            // 重置认证方式显示
            document.getElementById('passwordAuth').style.display = 'block';
            document.getElementById('keyAuth').style.display = 'none';
            document.getElementById('authPassword').checked = true;
            
            serverConfigModal.show();
        }
        
        // 显示配置服务器模态框
        function showConfigModal(serverId) {
            fetch('/api/multi-deploy/server/' + serverId)
                .then(response => response.json())
                .then(data => {
                    if (data.error) {
                        alert('获取服务器配置失败: ' + data.error);
                        return;
                    }
                    
                    const server = data.server;
                    document.getElementById('serverId').value = server.id;
                    document.getElementById('serverName').value = server.name;
                    document.getElementById('serverDomain').value = server.domain || '';
                    document.getElementById('serverHost').value = server.host;
                    document.getElementById('serverPort').value = server.port;
                    document.getElementById('serverUsername').value = server.username;
                    document.getElementById('serverPassword').value = '';
                    document.getElementById('serverKeyPath').value = server.key_path || '';
                    document.getElementById('serverRemotePath').value = server.remote_path;
                    document.getElementById('serverEnabled').checked = server.enabled;
                    
                    if (server.key_path) {
                        document.getElementById('authKey').checked = true;
                        document.getElementById('keyAuth').style.display = 'block';
                        document.getElementById('passwordAuth').style.display = 'none';
                    } else {
                        document.getElementById('authPassword').checked = true;
                        document.getElementById('passwordAuth').style.display = 'block';
                        document.getElementById('keyAuth').style.display = 'none';
                    }
                    
                    serverConfigModal.show();
                })
                .catch(error => {
                    alert('获取服务器配置失败: ' + error.message);
                });
        }
        
        // 保存服务器配置
        function saveServerConfig() {
            const formData = new FormData(document.getElementById('serverConfigForm'));
            const serverData = {
                name: formData.get('name'),
                domain: formData.get('domain'),
                host: formData.get('host'),
                port: parseInt(formData.get('port')),
                username: formData.get('username'),
                remote_path: formData.get('remote_path'),
                enabled: formData.get('enabled') === 'on'
            };
            
            const authMethod = formData.get('auth_method');
            if (authMethod === 'password') {
                serverData.password = formData.get('password');
            } else {
                serverData.key_path = formData.get('key_path');
            }
            
            const serverId = formData.get('server_id');
            const url = serverId ? '/api/multi-deploy/server/' + serverId : '/api/multi-deploy/server';
            const method = serverId ? 'PUT' : 'POST';
            
            fetch(url, {
                method: method,
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(serverData)
            })
            .then(response => response.json())
            .then(data => {
                if (data.error) {
                    alert('保存失败: ' + data.error);
                    return;
                }
                
                alert(data.message);
                serverConfigModal.hide();
                location.reload();
            })
            .catch(error => {
                alert('保存失败: ' + error.message);
            });
        }
        
        // 删除服务器
        function deleteServer(serverId) {
            if (!confirm('确定要删除这个服务器配置吗？')) {
                return;
            }
            
            fetch('/api/multi-deploy/server/' + serverId, {
                method: 'DELETE'
            })
            .then(response => response.json())
            .then(data => {
                if (data.error) {
                    alert('删除失败: ' + data.error);
                    return;
                }
                
                alert(data.message);
                location.reload();
            })
            .catch(error => {
                alert('删除失败: ' + error.message);
            });
        }
        
        // 测试连接
        function testConnection(serverId) {
            const btn = event.target.closest('button');
            btn.disabled = true;
            btn.innerHTML = '<i class="bi bi-hourglass-split"></i>';
            
            fetch('/api/multi-deploy/test/' + serverId, {
                method: 'POST'
            })
            .then(response => response.json())
            .then(data => {
                if (data.error) {
                    alert('连接测试失败: ' + data.error);
                } else {
                    alert('连接测试成功');
                }
            })
            .catch(error => {
                alert('连接测试失败: ' + error.message);
            })
            .finally(() => {
                btn.disabled = false;
                btn.innerHTML = '<i class="bi bi-wifi"></i>';
            });
        }
        
        // 部署到服务器
        function deployServer(serverId, incremental = false) {
            const action = incremental ? 'incremental-deploy' : 'deploy';
            const message = incremental ? '正在增量部署...' : '正在全量部署...';
            updateServerAction(serverId, action, message);
        }
        
        // 构建并部署
        function buildAndDeploy(serverId, incremental = false) {
            const action = incremental ? 'incremental-build-deploy' : 'build-deploy';
            const message = incremental ? '正在构建和增量部署...' : '正在构建和全量部署...';
            updateServerAction(serverId, action, message);
        }
        
        // 暂停部署
        function pauseDeployment(serverId) {
            updateServerAction(serverId, 'pause', '正在暂停...');
        }
        
        // 继续部署
        function resumeDeployment(serverId) {
            updateServerAction(serverId, 'resume', '正在继续...');
        }
        
        // 停止部署
        function stopDeployment(serverId) {
            updateServerAction(serverId, 'stop', '正在停止...');
        }
        
        // 执行服务器操作
        function updateServerAction(serverId, action, message) {
            fetch('/api/multi-deploy/' + action + '/' + serverId, {
                method: 'POST'
            })
            .then(response => response.json())
            .then(data => {
                if (data.error) {
                    alert(action + '失败: ' + data.error);
                    return;
                }
                
                showNotification(data.message || message, 'info');
                
                // 开始轮询状态更新
                if (action === 'deploy' || action === 'build-deploy' || action === 'resume') {
                    // WebSocket会处理实时更新
                }
            })
            .catch(error => {
                alert(action + '失败: ' + error.message);
            });
        }
        
        // 刷新服务器状态
        function refreshServerStatuses() {
            fetch('/api/multi-deploy/statuses')
                .then(response => response.json())
                .then(data => {
                    if (data.statuses) {
                        Object.keys(data.statuses).forEach(serverId => {
                            updateServerRow(serverId, data.statuses[serverId]);
                        });
                    }
                })
                .catch(error => {
                    console.error('刷新状态失败:', error);
                });
        }
        
        // 更新服务器行
        function updateServerRow(serverId, status) {
            const row = document.getElementById('server-row-' + serverId);
            if (!row) return;
            
            const statusIcon = row.querySelector('.status-icon');
            const statusBadge = row.querySelector('.status-badge');
            const progressContainer = row.querySelector('.progress-container');
            
            if (statusIcon) {
                statusIcon.className = 'status-icon status-' + status.status;
            }
            
            if (statusBadge) {
                statusBadge.className = 'badge status-badge bg-' + 
                    (status.status === 'success' ? 'success' : 
                     status.status === 'failed' ? 'danger' : 
                     status.status === 'building' ? 'warning' : 
                     status.status === 'deploying' ? 'info' : 
                     status.status === 'paused' ? 'warning' : 'secondary');
                statusBadge.textContent = getStatusText(status.status);
            }
            
            if (progressContainer && (status.status === 'building' || status.status === 'deploying')) {
                progressContainer.innerHTML = `
                    <div class="progress" style="height: 20px;">
                        <div class="progress-bar ${status.status === 'building' ? 'bg-warning' : 'bg-info'} progress-bar-striped progress-bar-animated" 
                             role="progressbar" style="width: ${status.progress}%" 
                             aria-valuenow="${status.progress}" aria-valuemin="0" aria-valuemax="100">
                            ${status.progress}%
                        </div>
                    </div>
                    ${status.speed ? `<div class="small text-muted mt-1"><i class="bi bi-speedometer2"></i> ${status.speed}</div>` : ''}
                `;
            } else if (progressContainer) {
                const iconClass = status.status === 'success' ? 'bi-check-circle-fill text-success' :
                                 status.status === 'failed' ? 'bi-x-circle-fill text-danger' : 
                                 'bi-dash-circle text-muted';
                progressContainer.innerHTML = `
                    <div class="text-center text-muted">
                        <i class="bi ${iconClass} fs-4"></i>
                    </div>
                `;
            }
        }
        
        // 获取状态文本
        function getStatusText(status) {
            const statusMap = {
                'idle': '空闲',
                'building': '构建中',
                'deploying': '部署中',
                'success': '成功',
                'failed': '失败',
                'paused': '已暂停'
            };
            return statusMap[status] || status;
        }
        
        // Hugo 构建功能
        function buildHugo() {
            const btn = event.target;
            btn.disabled = true;
            btn.innerHTML = '<i class="bi bi-hourglass-split"></i> 构建中...';
            
            addToLog('INFO: 开始Hugo构建...', 'info');
            
            fetch('/api/build-hugo', { method: 'POST' })
            .then(response => response.json())
            .then(data => {
                if (data.error) {
                    addToLog('ERROR: Hugo构建失败 - ' + data.error, 'error');
                    showNotification('构建失败: ' + data.error, 'error');
                } else {
                    addToLog('SUCCESS: Hugo构建成功', 'success');
                    showNotification('构建成功: ' + data.message, 'success');
                }
            })
            .catch(error => {
                addToLog('ERROR: Hugo构建失败 - ' + error.message, 'error');
                showNotification('构建失败: ' + error.message, 'error');
            })
            .finally(() => {
                btn.disabled = false;
                btn.innerHTML = '<i class="bi bi-play-fill"></i> 构建站点';
            });
        }

        function cleanAndBuild() {
            buildHugo(); // 简化为直接构建
        }
        
        // Hugo Serve 控制函数
        function startHugoServe() {
            fetch('/api/hugo-serve/start', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ port: 1313 })
            })
            .then(response => response.json())
            .then(data => {
                if (data.error) {
                    showNotification('启动失败: ' + data.error, 'error');
                } else {
                    showNotification(data.message, 'success');
                    setTimeout(() => location.reload(), 1000);
                }
            });
        }
        
        function restartHugoServe() {
            fetch('/api/hugo-serve/restart', { method: 'POST' })
            .then(response => response.json())
            .then(data => {
                if (data.error) {
                    showNotification('重启失败: ' + data.error, 'error');
                } else {
                    showNotification(data.message, 'success');
                    setTimeout(() => location.reload(), 1000);
                }
            });
        }
        
        function stopHugoServe() {
            fetch('/api/hugo-serve/stop', { method: 'POST' })
            .then(response => response.json())
            .then(data => {
                if (data.error) {
                    showNotification('停止失败: ' + data.error, 'error');
                } else {
                    showNotification(data.message, 'success');
                    setTimeout(() => location.reload(), 1000);
                }
            });
        }
        
        // 添加日志
        function addToLog(message, type = 'info') {
            const log = document.getElementById('deployLog');
            const timestamp = new Date().toLocaleTimeString();
            
            let colorClass = '';
            switch(type) {
                case 'success':
                    colorClass = 'text-success';
                    break;
                case 'error':
                    colorClass = 'text-danger';
                    break;
                case 'warning':
                    colorClass = 'text-warning';
                    break;
                default:
                    colorClass = 'text-info';
            }
            
            const logEntry = document.createElement('div');
            logEntry.className = colorClass;
            logEntry.textContent = `[${timestamp}] ${message}`;
            
            log.appendChild(logEntry);
            log.scrollTop = log.scrollHeight;
            
            saveLogToStorage(timestamp, message, type);
        }
        
        // 保存日志到本地存储
        function saveLogToStorage(timestamp, message, type) {
            try {
                let logs = JSON.parse(localStorage.getItem('hugo-deploy-logs') || '[]');
                logs.push({
                    timestamp: timestamp,
                    message: message,
                    type: type,
                    date: new Date().toISOString()
                });
                
                if (logs.length > 100) {
                    logs = logs.slice(-100);
                }
                
                localStorage.setItem('hugo-deploy-logs', JSON.stringify(logs));
            } catch (e) {
                console.warn('无法保存日志到本地存储:', e);
            }
        }
        
        // 从本地存储加载日志
        function loadLogsFromStorage() {
            try {
                const logs = JSON.parse(localStorage.getItem('hugo-deploy-logs') || '[]');
                const log = document.getElementById('deployLog');
                
                if (logs.length === 0) {
                    log.innerHTML = '<div class="text-muted">准备就绪，等待操作...</div>';
                    return;
                }
                
                log.innerHTML = '';
                logs.forEach(logItem => {
                    let colorClass = '';
                    switch(logItem.type) {
                        case 'success':
                            colorClass = 'text-success';
                            break;
                        case 'error':
                            colorClass = 'text-danger';
                            break;
                        case 'warning':
                            colorClass = 'text-warning';
                            break;
                        default:
                            colorClass = 'text-info';
                    }
                    
                    const logEntry = document.createElement('div');
                    logEntry.className = colorClass;
                    logEntry.textContent = `[${logItem.timestamp}] ${logItem.message}`;
                    log.appendChild(logEntry);
                });
                
                log.scrollTop = log.scrollHeight;
            } catch (e) {
                console.warn('无法加载日志从本地存储:', e);
            }
        }
        
        // 清空日志
        function clearLog() {
            document.getElementById('deployLog').innerHTML = 
                '<div class="text-muted">准备就绪，等待操作...</div>';
            try {
                localStorage.removeItem('hugo-deploy-logs');
            } catch (e) {
                console.warn('无法清除本地存储的日志:', e);
            }
        }