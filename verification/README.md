# 验证测试套件 (Verification Test Suite)

本目录包含用于验证 `secs4go` HSMS/GEM 实现正确性的集成测试和互操作性测试。

## 📋 测试文件概览

### Go 测试程序

| 文件 | 测试内容 | 用途 |
|------|---------|------|
| `test_host.go` | HSMS Host 基础连接 + S9 错误响应 | 验证 Host 能正确响应 Equipment 发送的无效消息 (S99F1, S1F99) |
| `test_timeout.go` | T3 超时处理 | 验证 Host 在 Equipment 不响应时触发 T3 超时并发送 S9F9 |
| `test_gem_s9.go` | GEM 层 S9 事件传播 | 验证 S9 错误事件从 HSMS 层正确传播到 GEM Handler |
| `test_p2_clock.go` | P2 时钟同步 | 验证 Host 发送 S2F17 (Get Time) 和 S2F31 (Set Time) 的功能 |
| `test_p2_alarm.go` | P2 报警管理 | 验证 Host 发送 S5F3 (Enable/Disable), S5F5 (List), S5F7 (List Enabled) 的功能 |

### Python 模拟器

| 文件 | 模拟角色 | 功能 |
|------|---------|------|
| `sim_equipment.py` | Equipment (被动模式) | 发送无效消息 (S99F1, S1F99) 触发 Host 的 S9 响应 |
| `sim_timeout.py` | Equipment (被动模式) | 不响应 S1F1，触发 Host 的 T3 超时；接收 S1F1 时发送 S9F3 测试 GEM 事件 |
| `sim_p2_clock.py` | Equipment (被动模式) | 支持 S2F17/F18 和 S2F31/F32，用于时钟同步验证 |
| `sim_p2_alarm.py` | Equipment (被动模式) | 支持 S5F3/F4, S5F5/F6, S5F7/F8，包含自定义递归解码器用于处理嵌套列表 |

### 辅助工具

| 文件 | 用途 |
|------|------|
| `check_s9f13.py` | 检查 `secsgem` 库是否实现 S9F13 |
| `explore_secsgem.py` | 快速探索 `secsgem` API |
| `inspect_handler.py` | 检查 `secsgem` GemEquipmentHandler 属性 |

---

## 🚀 运行测试

### 前置要求

1. **Python 3.x** + `secsgem` 库:
   ```bash
   pip install secsgem
   ```

2. **Go 1.20+**

3. **编译测试程序** (在项目根目录执行):
   ```bash
   go build -o verification/test_host.exe verification/test_host.go
   go build -o verification/test_timeout.exe verification/test_timeout.go
   go build -o verification/test_gem_s9.exe verification/test_gem_s9.go
   ```

---

## 📝 测试详情

### 1. S9 错误响应测试

**目标**: 验证 Host 能够正确发送 S9F3 (Unrecognized Stream) 和 S9F5 (Unrecognized Function)

**步骤**:

1. **启动 Equipment 模拟器** (终端 1):
   ```bash
   python verification/sim_equipment.py
   ```
   - 监听端口: `5010`
   - 行为: 连接建立 5 秒后，发送 `S99F1` 和 `S1F99`

2. **启动 Host 测试程序** (终端 2):
   ```bash
   verification/test_host.exe
   ```
   - 连接到: `127.0.0.1:5010`
   - 预期行为: 接收无效消息后自动回复 `S9F3` 和 `S9F5`

**验证结果**:
- **Python 终端**: 应显示收到 `S9F3` 和 `S9F5` 消息
- **Go 终端**: 应显示 "unrecognized stream/function" 日志

---

### 2. T3 超时测试

**目标**: 验证 Host 在等待响应超时后发送 S9F9 (Transaction Timer Timeout)

**步骤**:

1. **启动不响应的 Equipment 模拟器** (终端 1):
   ```bash
   python verification/sim_timeout.py
   ```
   - 监听端口: `5020`
   - 行为: 接收 `S1F1` 但 **不响应** (触发 T3 超时)

2. **启动 Timeout 测试程序** (终端 2):
   ```bash
   verification/test_timeout.exe
   ```
   - 连接到: `127.0.0.1:5020`
   - 发送 `S1F1` 并等待
   - 预期行为: 3 秒后 T3 超时，自动发送 `S9F9`

**验证结果**:
- **Python 终端**: 应显示收到 `S9F9` 消息
- **Go 终端**: 应显示 "T3 timeout" 和 "Sent S9F9" 日志

---

### 3. GEM 层 S9 事件测试

**目标**: 验证 S9 错误事件从 HSMS 协议层正确传播到 GEM Handler

**步骤**:

1. **启动响应式 Equipment 模拟器** (终端 1):
   ```bash
   python verification/sim_timeout.py
   ```
   - 监听端口: `5020`
   - 行为: 接收 `S1F1` 时，发送 **unsolicited S9F3** 消息

2. **启动 GEM S9 测试程序** (终端 2):
   ```bash
   verification/test_gem_s9.exe
   ```
   - 连接到: `127.0.0.1:5020`
   - 使用 `GemHandler` 订阅 `S9ErrorReceived` 事件
   - 发送 `S1F1` 触发模拟器发送 S9F3

**验证结果**:
- **Go 终端**: 
  - 应显示 "SUCCESS: Correctly received S9F3 event"
  - 确认 `GemHandler.Events().S9ErrorReceived` 被正确触发

---

### 4. P2 时钟和报警测试 (新增)

**目标**: 验证 GEM 时钟同步和报警管理功能的正确性。

**A. 时钟同步测试**:
1. **启动模拟器**: `python verification/sim_p2_clock.py` (端口 5030)
2. **运行测试**: `verification/test_p2_clock.exe`
3. **验证内容**:
   - Host 获取 Equipment 时间 (S2F17 -> S2F18)
   - Host 设置 Equipment 时间 (S2F31 -> S2F32)

**B. 报警管理测试**:
1. **启动模拟器**: `python verification/sim_p2_alarm.py` (端口 5240)
2. **运行测试**: `verification/test_p2_alarm.exe`
3. **验证内容**:
   - 列出所有报警 (S5F5 -> S5F6)
   - 启用/禁用报警 (S5F3 -> S5F4) - *注: Python 模拟器使用自定义递归解码器处理嵌套 S5F3 消息*
   - 列出已启用报警 (S5F7 -> S5F8)

---

## 🔧 端口配置

| 测试 | 默认端口 | 修改位置 |
|------|---------|---------|
| S9 错误响应 | 5010 | `sim_equipment.py` + `test_host.go` |
| T3 超时 | 5020 | `sim_timeout.py` + `test_timeout.go` |
| GEM S9 事件 | 5020 | `sim_timeout.py` + `test_gem_s9.go` |
| P2 时钟同步 | 5030 | `sim_p2_clock.py` + `test_p2_clock.go` |
| P2 报警管理 | 5240 | `sim_p2_alarm.py` + `test_p2_alarm.go` |

**修改方法**:
- Python: 找到 `port=5020`，修改端口号
- Go: 找到 `NewHsmsProtocol(..., 5020, ...)` 第二参数，修改端口号

---

## 📊 测试覆盖

- ✅ HSMS 连接建立 (Select/Deselect)
- ✅ S9F3 - Unrecognized Stream
- ✅ S9F5 - Unrecognized Function
- ✅ S9F9 - Transaction Timer Timeout (T3)
- ✅ GEM Handler 事件传播
- ✅ 自动重连机制 (隐式验证于所有测试)
- ⏸️ S9F1 - Unrecognized Device ID (未测试)
- ⏸️ S9F7 - Illegal Data (未测试)
- ⏸️ S9F11 - Data Too Long (未测试)
- ✅ S2F17/F18 - Date and Time Request
- ✅ S2F31/F32 - Date and Time Set
- ✅ S5F3/F4 - Enable/Disable Alarm
- ✅ S5F5/F6 - List All Alarms
- ✅ S5F7/F8 - List Enabled Alarms

---

## 🐛 故障排查

**问题**: Python 模拟器启动失败
- 检查是否已安装 `secsgem`: `pip list | grep secsgem`
- 检查端口是否被占用: `netstat -ano | findstr :5010`

**问题**: Go 程序编译失败
- 确保在项目根目录执行编译命令
- 运行 `go mod tidy` 更新依赖

**问题**: 连接超时
- 确认 Python 模拟器已启动且端口正确
- 检查防火墙是否阻止本地连接
- 确认 Go 程序的端口与 Python 模拟器一致
