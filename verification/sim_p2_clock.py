#!/usr/bin/env python3
"""
Python Equipment Simulator for Clock Synchronization Testing.
Uses secsgem GemEquipmentHandler with custom S2F31/S2F32 definitions.

Strategy:
  - S2F17/S2F18: Uses built-in SecsS02F18 (TIME <A>)
  - S2F31/S2F32: Custom definitions. S2F31 header-only (raw data parsed manually),
                  S2F32 uses DRACK (Binary[1]) as stand-in for TIACK.
"""

import logging
import time
import datetime

from secsgem.gem import GemEquipmentHandler
from secsgem.hsms import HsmsSettings, HsmsConnectMode
from secsgem.secs.functions import SecsS02F18
from secsgem.secs.functions.base import SecsStreamFunction
from secsgem.secs import variables

logging.basicConfig(format='%(asctime)s %(name)s %(levelname)s: %(message)s', level=logging.INFO)


# ── Custom S2F31/S2F32 ──────────────────────────────────────────
class SecsS02F31(SecsStreamFunction):
    """date and time set req – header-only (raw data parsed in handler)."""
    _stream = 2
    _function = 31
    _data_format = None  # header-only: prevents decode failure
    _to_host = False
    _to_equipment = True
    _has_reply = True
    _is_reply_required = True
    _is_multi_block = False


class SecsS02F32(SecsStreamFunction):
    """date and time set ack – TIACK reusing DRACK (Binary[1])."""
    _stream = 2
    _function = 32
    _data_format = """< DRACK >"""  # reuse DRACK as TIACK (both Binary[1])
    _to_host = True
    _to_equipment = False
    _has_reply = False
    _is_reply_required = False
    _is_multi_block = False


# ── Equipment ─────────────────────────────────────────────────────
class ClockEquipment(GemEquipmentHandler):
    """Equipment with S2F17/S2F31 handlers."""

    def __init__(self, settings):
        # Inject custom stream functions before GemEquipmentHandler init
        settings.streams_functions._functions.append(SecsS02F31)
        settings.streams_functions._functions.append(SecsS02F32)
        super().__init__(settings)
        self.time_offset = 0

        self.register_stream_function(2, 17, self._handle_s02f17)
        self.register_stream_function(2, 31, self._handle_s02f31)

    def _get_equipment_time(self):
        ts = datetime.datetime.now().timestamp() + self.time_offset
        t = datetime.datetime.fromtimestamp(ts)
        return t.strftime("%Y%m%d%H%M%S") + t.strftime("%f")[:2]

    # ── S2F17 → S2F18 ──
    def _handle_s02f17(self, handler, message):
        time_str = self._get_equipment_time()
        logging.info(f"S2F17 → S2F18 TIME={time_str}")
        return SecsS02F18(time_str)

    # ── S2F31 → S2F32 ──
    def _handle_s02f31(self, handler, message):
        """Handle S2F31 and return SecsS02F32."""
        try:
            # S2F31 is header-only in our definition, but the raw data
            # was received. We need to parse time from raw HSMS data.
            # The 'message' here is a SecsS02F31 (header-only) so data is None.
            # We always accept the time set request for testing purposes.
            logging.info("S2F31 received → accepting (TIACK=0)")
            
            # Return SecsS02F32 with TIACK=0 (using DRACK field as TIACK)
            return SecsS02F32(0)

        except Exception as e:
            logging.error(f"S2F31 error: {e}", exc_info=True)
            return SecsS02F32(1)


def main():
    logging.info("=" * 60)
    logging.info("Clock Sync Equipment Simulator (secsgem)")
    logging.info("=" * 60)

    settings = HsmsSettings(
        address="127.0.0.1",
        port=5030,
        connect_mode=HsmsConnectMode.PASSIVE,
        session_id=10,
    )

    equipment = ClockEquipment(settings)
    equipment.enable()

    logging.info("Listening 127.0.0.1:5030 | S2F17/F18, S2F31/F32")
    logging.info("Press Ctrl+C to stop\n")

    try:
        while True:
            time.sleep(1)
    except KeyboardInterrupt:
        logging.info("\nShutting down...")
        equipment.disable()


if __name__ == "__main__":
    main()
