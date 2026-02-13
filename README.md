### Introduction

Simple Golang SECS/GEM implementation.

* [x] SECS-II
* [x] HSMS, Not supported HSMS-GS
* [x] GEM (handshake, communication state, Are You There)

### Quick Start

```
opts := gem.Options{
    Protocol:   hsms.NewHsmsProtocol("127.0.0.1", 5000, true, 0xFFFF, "sample"),
    DeviceType: gem.DeviceHost,
}
handler, err := gem.NewGemHandler(opts)
if err != nil {
    log.Fatal(err)
}
handler.Enable()
if handler.WaitForCommunicating(30 * time.Second) {
    log.Println("GEM session established")
}
```

### Acknowledgements

* [funny/link]( https://github.com/funny/link): Go Networking Scaffold
* [wolimst/lib-secs2-hsms-go](https://github.com/wolimst/lib-secs2-hsms-go): SECS-II/HSMS/SML Data Parser
* reference doc: [scesgem](https://github.com/bparzella/secsgem), [python doc](https://secsgem.readthedocs.io/en/latest/index.html)

### Other

The third-party library has been heavily modified, so it is directly imported and used in the project.

This library is only a portion, completing the most fundamental part. Use it with caution.

### GEM Capabilities

- Alarm reporting: register alarms via \\RegisterAlarm\\ and trigger S5F1/S5F2 with \\RaiseAlarm\\ / \\ClearAlarm\\.
- Remote command support: host calls `SendRemoteCommand` (S2F41/42, returns `RemoteCommandResult`) while equipment hooks `SetRemoteCommandHandler`.

### Logging Configuration

The library uses a plugin-style logger. By default, it is completely silent (Zero Mental Burden). You can configure it for:

1.  **Standard Console Logging**:
    ```go
    opts := gem.Options{
        // ...
        Logger: gem.NewStdLogger(os.Stdout, "gem: "),
    }
    ```

2.  **Structured File Logging (with Rotation)**:
    Requires `go.uber.org/zap` and `gopkg.in/natefinch/lumberjack.v2`.
    ```go
    // 1. App Logger (Info/Error/Debug messages)
    appLogger := gem.NewZapLogger(gem.ZapLoggerOptions{
        LogFile: "app.log",
        MaxSize: 10, // MB
        MaxBackups: 5,
        DebugLevel: true,
    })

    opts := gem.Options{
        // ...
        Logger: appLogger,

        // 2. Protocol Trace Logging (SML/Binary)
        Logging: gem.LoggingOptions{
            Enabled:    true,
            LogFile:    "protocol.log", // Separate rotation for SML traces
            MaxSize:    100, // MB
            MaxBackups: 10,
        },
    }
    ```

#### Parameter Reference

**App Logger (`gem.ZapLoggerOptions`)**
| Parameter | Type | Description | Default |
| :--- | :--- | :--- | :--- |
| `LogFile` | string | Log file path. If empty, logs to stdout. | "" |
| `MaxSize` | int | Max size (MB) before rotation. | 100 |
| `MaxBackups` | int | Max number of old files to keep. | - |
| `MaxAge` | int | Max days to keep old files. | - |
| `Compress` | bool | Gzip compress old files. | false |
| `DebugLevel` | bool | Enable DEBUG level logs. | false |
| `Console` | bool | Print to stdout even if LogFile is set. | false |

**Protocol Logger (`gem.LoggingOptions`)**
| Parameter | Type | Description | Default |
| :--- | :--- | :--- | :--- |
| `Enabled` | bool | Enable SML/Binary protocol tracing. | false |
| `LogFile` | string | Log file path. If empty, logs to stderr. | "" |
| `MaxSize` | int | Max size (MB) before rotation. | 100 |
| `MaxBackups` | int | Max number of old files to keep. | - |
| `MaxAge` | int | Max days to keep old files. | - |
| `Compress` | bool | Gzip compress old files. | false |

### Error Handling

The library implements robust error handling and reporting:
-   **Automatic S9 Generation**: Automatically sends S9 'Error' messages for protocol violations:
    -   S9F1 (Unrecognized Device ID)
    -   S9F3 (Unrecognized Stream)
    -   S9F5 (Unrecognized Function)
    -   S9F7 (Illegal Data)
    -   S9F9 (Transaction Timer Timeout) - triggered by T3 expiration
    -   S9F11 (Data Too Long)
-   **Event Notification**: The `GemHandler` exposes an `S9ErrorReceived` event to notify the application of these errors.
    ```go
    handler.Events().S9ErrorReceived.AddCallback(func(data map[string]interface{}) {
        if info, ok := data["error"].(*hsms.S9ErrorInfo); ok {
             log.Printf("S9 Error: %s", info.ErrorText)
        }
    })
    ```

### Integration

- A reference integration test lives in \\gem/handler_integration_test.go\\ (skipped by default). Remove the \\	.Skip\\ line and run \\go test ./gem -run TestGemHandlerIntegration -count=1\\ to exercise a full Go host/equipment loopback.
- To cross-check with the original Python implementation, run the Python sample host (e.g. \\samples/gem/host.py\\) and start the Go equipment sample from the Quick Start with \\DeviceType: gem.DeviceEquipment\\. Both sides exchange S1/S2/S5 flows once communication is established.