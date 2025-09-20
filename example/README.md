# secs4go Examples

The example directory now mirrors the secsgem Java sample layout. Each program isolates a GEM capability so you can validate behaviour quickly.

| Example | Description | Run command |
|---------|-------------|-------------|
| example1_passive_equipment | Passive HSMS equipment built on the Go GemHandler. Registers status/data variables, a remote command, and periodically fires CEID 3001. | go run ./example/example1_passive_equipment --port 5000 |
| example2_active_host | Active HSMS host that connects to the passive equipment and exercises status polling, dynamic reports, remote commands, and process programs. | go run ./example/example2_active_host --scenario all |
| gem_pytest | Interoperability harness that talks to the Python secsgem reference implementation. | go run ./example/gem_pytest --role host |
| geminterop | Combined host/equipment playground retained for backwards compatibility. | go run ./example/geminterop --mode host |

Shared identifiers between host and equipment samples:

- Status variable SVID=1001 (temperature) and data variable VID=2001 (pressure).
- Collection event CEID=3001 that emits status snapshots.
- Remote command START that triggers an S6F11 once report links are active.
- Process program PPID="SAMPLE" for upload/download demonstrations.

## Quick start

1. Start the equipment sample:

       go run ./example/example1_passive_equipment --port 5000

2. From another terminal, run a host scenario:

       go run ./example/example2_active_host --scenario all

Both programs enable SML logging, so you can correlate HSMS traffic with GEM behaviour.
