# Using secs4go for GEM Host and Equipment Development

This guide walks through the essential concepts required to build a GEM-compliant host or equipment application with the `secs4go` project.

## 1. Project Layout

```
secs4go/
  gem/               Core GEM handler implementation
  hsms/              HSMS transport layer
  example/           Ready-to-run sample apps
  lib-secs2-hsms-go/ SECS-II AST and HSMS parsers
samples/
  gem_equipment.py   Python equipment sample (secsgem)
  gem_host.py        Python host sample (secsgem)
```

## 2. HSMS Connection Basics

Create an `hsms.HsmsProtocol` for the peer you want to connect to. The constructor accepts:

- `address` / `port` of the remote peer
- `active` (`true` for host/client, `false` for equipment/server)
- HSMS `sessionID`
- `name` (used only for logging)

```go
protocol := hsms.NewHsmsProtocol("127.0.0.1", 5000, false, 0x200, "eqp")
protocol.Timeouts().SetLinktest(30) // optional overrides
```

`sessionID` is the application session used for SECS-II data once the connection is SELECTED. Control packets such as `select.req` may legally use the wildcard `0xFFFF`; the Go stack automatically adopts the negotiated session in that case.

## 3. Creating a GEM Handler

Wrap the HSMS layer with a GEM handler:

```go
handler, err := gem.NewGemHandler(gem.Options{
    Protocol:   protocol,
    DeviceType: gem.DeviceEquipment,
    MDLN:       "demo-eqp",
    SOFTREV:    "1.0.0",
    Logging: gem.LoggingOptions{ // optional, see section 7
        Enabled:                true,
        Mode:                   hsms.LoggingModeSML,
        IncludeControlMessages: false,
    },
})
if err != nil {
    log.Fatalf("create handler: %v", err)
}
```

Key options:

- `DeviceType`: `gem.DeviceHost` or `gem.DeviceEquipment`
- Equipment-only fields: `MDLN`, `SOFTREV`
- `EstablishCommunicationWait`, `InitialControlState`, `InitialOnlineMode`
- `Logging`: controls on-the-wire diagnostics (see section 7)

Activate the handler with `handler.Enable()` and optionally wait for `handler.WaitForCommunicating(timeout)`.

## 4. Equipment Development Workflow

1. **Register status/data variables**
   ```go
   sv, _ := gem.NewStatusVariable(1001, "Temperature", "C",
       gem.WithStatusValueProvider(func() (ast.ItemNode, error) {
           return ast.NewUintNode(4, 25), nil
       }),
   )
   handler.RegisterStatusVariable(sv)
   ```
2. **Register equipment constants** 每 supply min/max/default values and update callbacks as needed.
3. **Declare collection events & reports** 每 `handler.RegisterCollectionEvent` and `handler.RegisterDataVariable` define CEIDs and VIDs the host can subscribe to.
4. **Remote commands** 每 install a handler via `handler.SetRemoteCommandHandler` to process S2F41 requests.
5. **Process programs** 每 store recipes with `handler.RegisterProcessProgram` or override upload/download callbacks.
6. **Alarms** 每 register with `handler.RegisterAlarm` and raise via `handler.RaiseAlarm`.

The equipment example (`runPythonHostEquipment` in `example/gem_pytest/main.go`) demonstrates these capabilities, including a CEID that reports both status and data variables.

## 5. Host Development Workflow

1. **Query metadata & values** 每 use `RequestStatusVariableInfo`, `RequestStatusVariables`, `RequestEquipmentConstantInfo`, etc.
2. **Define and link reports** 每 always clear stale definitions first:
   ```go
   handler.DefineReports() // S2F33 clear
   handler.DefineReports(gem.ReportDefinitionRequest{
       ReportID: 4001,
       VIDs:     []interface{}{1001, 2001},
   })
   handler.LinkEventReports(gem.EventReportLinkRequest{CEID: 3001, ReportIDs: []interface{}{4001}})
   handler.EnableEventReports(true, 3001)
   ```
3. **Collection event snapshots** 每 `handler.RequestCollectionEventReport(ceid)` wraps S6F15/S6F16.
4. **Process program upload/download** 每 `UploadProcessProgram` (S7F3) and `RequestProcessProgram` (S7F5). Handle non-zero ACKs gracefully.
5. **Remote commands** 每 `handler.SendRemoteCommand("START", params)` returns HCACK; `0` is success, `4` means ※acknowledged, finish later§.

## 6. Handling Callbacks and Extensions

- **Custom stream handlers** 每 `handler.RegisterStreamFunctionHandler(stream, function, fn)` to intercept specific SECS-II messages, `nil` to remove.
- **Default stream handler** 每 `handler.RegisterDefaultStreamHandler(fn)` installs a fallback.
- **Event subscriptions** 每 `handler.Events()` exposes typed callbacks, e.g. `EventReportReceived`.
- **Timeout tuning** 每 `protocol.Timeouts()` exposes setters for T3/T5/T6/T7/T8 and linktest intervals.

## 7. Logging HSMS/GEM Traffic

Detailed wire logging can be toggled through `gem.LoggingOptions`:

```go
handler, err := gem.NewGemHandler(gem.Options{
    Protocol:   protocol,
    DeviceType: gem.DeviceEquipment,
    Logging: gem.LoggingOptions{
        Enabled:                true,
        Mode:                   hsms.LoggingModeBoth, // SML + hex
        IncludeControlMessages: false,                // hide select/linktest
    },
})
```

- `LoggingModeSML` prints human-readable SML (default when enabled).
- `LoggingModeBinary` prints hexadecimal HSMS frames.
- `LoggingModeBoth` prints SML followed by the hex dump.
- Provide a custom `io.Writer` via `LoggingOptions.Writer` to redirect output.
- Adjust behaviour at runtime with `protocol.ConfigureLogging`.

## 8. Interoperability Checks

Run the built-in tests:

```
cd secs4go
go test ./...
```

The `gem` package integration test exercises host?equipment flows.

## 9. Common Pitfalls & Tips

- **Report redefinition** 每 equipment persists S2F33 definitions; clear them before redefining.
- **Session ID** 每 control packets may use `0xFFFF`; the Go stack adapts automatically.
- **ACK/HCACK handling** 每 non-zero codes are normal; design retries or fallbacks instead of panicking.
- **Linktest** 每 enable linktest for long-lived connections.

## 10. Getting Started Quickly

1. Go-only loopback: `go run ./example/example1_passive_equipment` and `go run ./example/example2_active_host --scenario all`.
2. Interop with Python (secsgem):
   - Equipment: `python samples/gem_equipment.py`
   - Host: `go run example/gem_pytest/main.go --role host`
3. Observe the logs (optionally with logging enabled) to see each GEM capability exercised.

From here, adapt the examples by registering your own data sources, process program handlers, or remote commands to build production-ready equipment or host applications.
