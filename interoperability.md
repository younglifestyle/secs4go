# GEM Interoperability Quickstart

This guide shows how to exercise the new Go GEM capabilities (status variables, data values, collection events, equipment constants and process programs) and how to interoperate with the original Python `secsgem` implementation.

## Prerequisites

* Go 1.21+
* Python 3.10+ with `secsgem` installed (`pip install secsgem`)
* Two terminal windows – one for the Go process, one for the Python process

## Running the Go sample equipment/host

The repository now ships with an interoperability helper in `secs4go/example/geminterop`.

### Go equipment

```bash
go run ./secs4go/example/geminterop --mode equipment --addr 127.0.0.1 --port 5000 --session 0x100
```

The equipment registers:

* Status Variable 1001 (`Temperature`)
* Data Variable 2001 (`Pressure`)
* Collection Event 3001
* Process Program `SAMPLE`

Every 10 seconds it triggers CEID 3001 if a host has linked reports.

### Go host

```bash
go run ./secs4go/example/geminterop --mode host --addr 127.0.0.1 --port 5000 --session 0x100
```

The host demonstrates:

1. Defining report 4001 with SVID 1001 and VID 2001 (`DefineReports`)
2. Linking CEID 3001 to report 4001 (`LinkEventReports`)
3. Enabling event reporting (`EnableEventReports`)
4. Uploading a process program (`UploadProcessProgram`)
5. Requesting a stored program (`RequestProcessProgram`)
6. Receiving live S6F11 notifications (printed to the console)

Quit with <kbd>Ctrl</kbd>+<kbd>C</kbd>.

## Interoperating with Python `secsgem`

The same sample can be used to validate Go↔Python compatibility. Start either implementation in host or equipment mode and let the other side be provided by Python.

### Python equipment with Go host

```bash
python -m secsgem.examples.gem_equipment --port 5000
```

In another terminal run the Go host sample as shown earlier. You should see `S6F11` notifications in the Go host log while Python prints the incoming S7F3 upload requests.

### Go equipment with Python host

```bash
python -m secsgem.examples.gem_host --port 5000
```

Let the Go equipment sample run. The Python host will define reports and request process programs; the Go equipment will log incoming requests and continue providing S6F11 data.

## Automated verification

The Go test suite exercises all GEM features with an in-process host/equipment pair. Run:

```bash
go test ./secs4go/gem
```

This validates:

* Status variable and equipment constant exchange
* Data variables and collection event reporting
* Report definition/linking and enable/disable flows
* Process program upload/download acknowledgements

For Python compatibility, run the corresponding `secsgem` tests:

```bash
pytest tests/test_gem_equipment_handler.py -k "status or report or process"
```

With both sets passing and the interop sample exchanging S6F11/S7F3 traffic, GEM functionality can be considered verified in both directions.
