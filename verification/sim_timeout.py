
import logging
import time
import sys
import threading
import socket

from secsgem.gem import GemEquipmentHandler
from secsgem.hsms import HsmsSettings, HsmsConnectMode, HsmsMessage, HsmsStreamFunctionHeader
from secsgem.secs.functions import SecsStreamFunction
from secsgem.secs.variables import Binary

# Define custom S9F3
class S09F03(SecsStreamFunction):
    _stream = 9
    _function = 3
    _wait_for_reply = False
    _to_host = True
    _to_equipment = False
    _data_format = Binary

# Configure logging
logging.basicConfig(
    format='%(asctime)s %(name)s %(levelname)s: %(message)s',
    level=logging.INFO
)

class TimeoutTestEquipment(GemEquipmentHandler):
    def __init__(self, settings):
        super().__init__(settings, "EQUIPMENT", "1.0.0")
        self.received_s9f9 = False
        self.s9_lock = threading.Lock()
        self.delay_responses = False
        
    def _on_s01f01(self, handler, packet):
        # Trigger sending S9F3 to Host for verification of Phase 5
        logging.info("Received S1F1. Sending unsolicited S9F3 to Host for testing...")
        
        # Construct fake header for the S9F3 payload
        header_bytes = b'\x00' * 10 
        
        # Use our custom S9F3 class
        s9f3 = S09F03(header_bytes)
        self.send_stream_function(s9f3)
        
        return super()._on_s01f01(handler, packet)

    def on_message_received(self, message):
        try:
            stream = message.header.stream
            function = message.header.function
            
            if stream == 100:
                logging.info(f"Received S{stream}F{function} (Unrecognized Stream) - Sending S9F3")
                # Construct Payload (Header)
                # Just use 10 bytes of dummy data if we can't easily get header bytes
                # Or try to encode header
                header_bytes = b'\x00' * 10
                if hasattr(message.header, 'encode'):
                    header_bytes = message.header.encode()
                
                s9f3 = S09F03(header_bytes)
                self.send_stream_function(s9f3)
                return

            if stream == 9:
                with self.s9_lock:
                    if function == 3:
                        logging.info("Received S9F3")
                    elif function == 9:
                        logging.info("✓ Received S9F9 (Transaction Timer Timeout)")
                        self.received_s9f9 = True
                    else:
                        logging.info(f"Received S9F{function} (Unexpected in timeout test)")
                
        except Exception as e:
            logging.error(f"Error handling message: {e}")
        
        # Call base handler
        super().on_message_received(message)

def main():
    # Use port 5020 for timeout test
    settings = HsmsSettings(
        address="127.0.0.1",
        port=5020,
        connect_mode=HsmsConnectMode.PASSIVE,
        session_id=10
    )

    equipment = TimeoutTestEquipment(settings)
    equipment.enable()
    
    print("Timeout Test Equipment started on port 5020.")
    print("Waiting for Host connection and S1F1 (Are You There)...")
    
    # Enable response delay after a brief moment to allow connection
    equipment.delay_responses = True
    
    # Wait loop
    start_time = time.time()
    try:
        while time.time() - start_time < 30:
            time.sleep(1)
            if equipment.received_s9f9:
                print("SUCCESS: Received S9F9 Transaction Timer Timeout!")
                sys.exit(0)
                
        print("FAILURE: Timed out waiting for S9F9.")
        sys.exit(1)

    except KeyboardInterrupt:
        equipment.disable()

if __name__ == "__main__":
    main()
