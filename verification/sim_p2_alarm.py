#!/usr/bin/env python3
"""
Python Equipment Simulator for P2 Alarm Management Testing.
Final ultra-robust version with recursive manual decoding.
"""

import logging
import time
import secsgem.secs.functions.streams_functions

logging.basicConfig(format='%(asctime)s %(levelname)s: %(message)s', level=logging.INFO)

# ── Global Patch to bypass S5F3 decoding ───────────────────────
# We want raw message in handler so we can decode it manually
original_decode = secsgem.secs.functions.streams_functions.StreamsFunctions.decode
def patched_decode(self, message):
    if message.header.stream == 5 and message.header.function == 3:
        return message
    return original_decode(self, message)
secsgem.secs.functions.streams_functions.StreamsFunctions.decode = patched_decode

from secsgem.gem import GemEquipmentHandler
from secsgem.hsms import HsmsSettings, HsmsConnectMode
from secsgem.secs import variables

class AlarmEquipment(GemEquipmentHandler):
    def __init__(self, settings):
        super().__init__(settings)
        self.alarm_db = {
            1001: {"text": "HighTemp", "enabled": True},
            1002: {"text": "LowPress", "enabled": True},
            1003: {"text": "DoorOpen", "enabled": True},
        }
        self.register_stream_function(5, 3, self._handle_s05f03)
        self.register_stream_function(5, 5, self._handle_s05f05)
        self.register_stream_function(5, 7, self._handle_s05f07)

    def recursive_decode(self, data, start=0):
        """Recursively decode SECS data (handles Lists manually, Scalars via Dynamic)."""
        byte = data[start]
        format_code = (byte & 0xFC) >> 2
        
        if format_code == 0: # List
            len_bytes = byte & 0x03
            current = start + 1
            length = 0
            for _ in range(len_bytes):
                length = (length << 8) + data[current]
                current += 1
            
            items = []
            for _ in range(length):
                val, current = self.recursive_decode(data, current)
                items.append(val)
            return items, current
        else:
            # Use Dynamic([]) to allow all supported types
            d = variables.Dynamic([]) 
            new_pos = d.decode(data, start)
            return d.get(), new_pos

    def _handle_s05f03(self, handler, message):
        try:
            # message is raw HsmsMessage due to patched_decode
            logging.info("Decoding raw HsmsMessage for S5F3 using recursive_decode")
            
            decoded_data, _ = self.recursive_decode(message.data)
            logging.info(f"Decoded data: {decoded_data}")
            
            # Expected: [aled_val, [alid1, alid2...]]
            # aled_val might be bytes/list/int depending on library.
            # Usually Binary -> bytes or list of int if count>1?
            # Let's inspect structure safely.
            
            aled = decoded_data[0]
            ids = decoded_data[1]
            
            # Determine enable/disable
            # ALED is Binary. 1 byte. 0x80 = Enable (128).
            is_enable = False
            if isinstance(aled, (bytes, bytearray)):
                is_enable = (aled[0] == 128)
            elif isinstance(aled, list):
                 is_enable = (aled[0] == 128)
            elif isinstance(aled, int):
                 is_enable = (aled == 128)
            
            logging.info(f"S5F3 parsed: enable={is_enable}, ids={ids}")
            
            ackc5 = 0
            for alid in ids:
                target_id = int(alid)
                if target_id in self.alarm_db:
                    self.alarm_db[target_id]["enabled"] = is_enable
                    logging.info(f"Alarm {target_id} enabled set to {is_enable}")
                else:
                    logging.warning(f"S5F3 unknown ALID: {target_id}")
                    ackc5 = 1
            
            from secsgem.secs.functions import SecsS05F04
            return SecsS05F04(ackc5)
        except Exception as e:
            logging.error(f"S5F3 error: {e}", exc_info=True)
            from secsgem.secs.functions import SecsS05F04
            return SecsS05F04(1)

    def _handle_s05f05(self, handler, message):
        from secsgem.secs.functions import SecsS05F06
        data = [{"ALCD": 0x40 if v["enabled"] else 0, "ALID": k, "ALTX": v["text"]} 
                for k, v in sorted(self.alarm_db.items())]
        return SecsS05F06(data)

    def _handle_s05f07(self, handler, message):
        from secsgem.secs.functions import SecsS05F08
        data = [{"ALCD": 0x40, "ALID": k, "ALTX": v["text"]} 
                for k, v in sorted(self.alarm_db.items()) if v["enabled"]]
        return SecsS05F08(data)

def main():
    settings = HsmsSettings(address="127.0.0.1", port=5240, connect_mode=HsmsConnectMode.PASSIVE, session_id=10)
    settings.timeouts.t3 = 30
    equipment = AlarmEquipment(settings)
    equipment.enable()
    logging.info("Listening on 5240 (Recursive Decode mode)")
    try:
        while True: time.sleep(1)
    except KeyboardInterrupt:
        equipment.disable()

if __name__ == "__main__":
    main()
