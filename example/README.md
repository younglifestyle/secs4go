# SECSGEM 示例 (Examples)

这个目录包含演示如何使用 `secs4go` 库构建 Host（主机）和 Equipment（设备）应用程序的示例。

## 前置条件

- Go 1.20+
- Python 3.10+ (用于运行参考模拟器)
- `secsgem` Python 库 (`pip install secsgem`)

## 示例 1: Passive Equipment (Go) + Active Host (Python)

此示例演示了一个基于 Go 的设备程序，它监听连接（被动模式）。我们使用一个 Python 脚本来模拟主机（主动模式）。

### 测试功能点：
1.  **建立连接 (S1F13)**: Go 设备作为 Server 等待连接，Python 主机作为 Client 发起连接。
2.  **状态变量查询 (S1F11/S1F12)**: Python 主机请求变量列表，Go 设备返回定义的 SVID。
3.  **状态数据快照 (S1F3/S1F4)**: Python 主机请求特定 SVID 的值，Go 设备返回实时数据（模拟温度和压力）。
4.  **动态报告 (Dynamic Reports)**:
    *   **定义报告 (S2F33)**: 主机定义 Report ID (RPTID) 和包含的变量 (VID)。
    *   **链接事件 (S2F35)**: 主机将 Report ID 链接到事件 ID (CEID)。
    *   **启用事件 (S2F37)**: 主机启用特定事件报告。
5.  **事件上报 (S6F11)**: Go 设备在收到远程命令后，模拟触发事件并上报包含数据的 S6F11 消息。
6.  **远程命令 (S2F41)**: Python 主机发送 "START" 命令，Go 设备接收并在后台触发事件。

### 1. 启动设备 (Go)
设备监听端口 15000 (Passive mode)。

```bash
# 在项目根目录下运行
go run ./example/example1_passive_equipment/main.go
# 输出: equipment listening on 127.0.0.1:15000 session=0x2
```

### 2. 启动主机 (Python)
Python 脚本连接到设备并执行完整的 SECS/GEM 流程。

```bash
# 在项目根目录下运行
python example/pythonExample/samples/gem_host2.py
```

---

## 示例 2: Active Host (Go) + Passive Equipment (Python)

此示例演示了一个基于 Go 的主机程序，它主动连接到设备。我们使用一个 Python 脚本来模拟设备（被动模式）。

### 测试功能点：
1.  **建立连接 (S1F13)**: Go 主机主动连接 Python 模拟器。
2.  **状态变量发现 (S1F11)**: Go 主机请求设备的所有状态变量列表。
3.  **状态数据轮询 (S1F3)**: Go 主机定期请求特定 SVID（温度、压力、RunID）的值。
4.  **SVID 兼容性测试**: 验证 Go 主机能否正确处理 Python 模拟器返回的标准 SVID (如 Clock) 和自定义 SVID。

### 1. 启动设备模拟器 (Python)
Python 模拟器监听端口 15000 (Passive mode)。

```bash
# 在项目根目录下运行
python example/pythonExample/samples/gem_eqp2.py
```

### 2. 启动主机 (Go)
主机连接到设备并执行状态轮询场景。

```bash
# 在项目根目录下运行
go run ./example/example2_active_host/main.go -scenario status
# 输出: host in COMMUNICATING state ... SVID=... VALUE=...
```