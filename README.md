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

### Other

The third-party library has been heavily modified, so it is directly imported and used in the project.

This library is only a portion, completing the most fundamental part. Use it with caution.

### GEM Capabilities

- Alarm reporting: register alarms via \\RegisterAlarm\\ and trigger S5F1/S5F2 with \\RaiseAlarm\\ / \\ClearAlarm\\.
- Remote command support: host calls \\SendRemoteCommand\\ (S2F41/42) while equipment hooks \\SetRemoteCommandHandler\\.

### Integration

- A reference integration test lives in \\gem/handler_integration_test.go\\ (skipped by default). Remove the \\	.Skip\\ line and run \\go test ./gem -run TestGemHandlerIntegration -count=1\\ to exercise a full Go host/equipment loopback.
- To cross-check with the original Python implementation, run the Python sample host (e.g. \\samples/gem/host.py\\) and start the Go equipment sample from the Quick Start with \\DeviceType: gem.DeviceEquipment\\. Both sides exchange S1/S2/S5 flows once communication is established.
