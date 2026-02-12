
import logging
import time
import sys
import threading

from secsgem.gem import GemEquipmentHandler
from secsgem.hsms import HsmsSettings, HsmsConnectMode, HsmsMessage, HsmsStreamFunctionHeader

# Configure logging
logging.basicConfig(
    format='%(asctime)s %(name)s %(levelname)s: %(message)s',
    level=logging.INFO
)

class ValidatedEquipment(GemEquipmentHandler):
    def __init__(self, settings):
        super().__init__(settings, "EQUIPMENT", "1.0.0")
        self.received_s9f3 = False
        self.received_s9f5 = False
        self.s9_lock = threading.Lock()

    def on_message_received(self, message):
        # Override to inspect messages
        try:
            stream = message.header.stream
            function = message.header.function
            logging.info(f"Received S{stream}F{function}")
            
            if stream == 9:
                with self.s9_lock:
                    if function == 3:
                        logging.info("✓ Received S9F3 (Unrecognized Stream)")
                        self.received_s9f3 = True
                    elif function == 5:
                        logging.info("✓ Received S9F5 (Unrecognized Function)")
                        self.received_s9f5 = True
                    else:
                        logging.info(f"Received S9F{function} (Unexpected)")
                        
        except Exception as e:
            logging.error(f"Error handling message: {e}")
        
        # Call base handler to maintain GEM state
        super().on_message_received(message)

def main():
    settings = HsmsSettings(
        address="127.0.0.1",
        port=5010,
        connect_mode=HsmsConnectMode.PASSIVE,
        session_id=10
    )

    equipment = ValidatedEquipment(settings)
    equipment.enable()
    
    print("Equipment started on port 5010. Waiting for Host connection...")
    
    # Wait for connection
    try:
        # Simple wait loop for connection
        # In secsgem, connection status is handled internally
        # We can check internal state or just wait
        wait_count = 0
        while wait_count < 30:
            # We assume connection will happen. 
            # Ideally exposure of connection state would be better but keeping it simple.
            time.sleep(1)
            wait_count += 1
            # There isn't an easy public "is_connected" property in base handler without digging
            # but usually 5-10 seconds is enough if host is running
            if wait_count == 5:
                print("Attempting to send invalid messages...")
                print(f"Equipment attributes: {dir(equipment)}")
                
                # 1. Send Unknown Stream (S99F1)
                try:
                    # Construct header for S99F1
                    header = HsmsStreamFunctionHeader(
                        system=1001,
                        stream=99,
                        function=1,
                        session_id=10,
                        require_response=True
                    )
                    
                    # Create message with empty body
                    msg = HsmsMessage(header=header, data=b'')
                    
                    # Send using the protocol's send_message method
                    # equipment is a Protocol/Handler instance which usually has send_message
                    # if sending HsmsMessage directly, we might need access to connection
                    # But Protocol usually has send_message(msg).
                    
                    # equipment.send_message(msg) might expect a SecsMessage object if high level.
                    # But low level send_message often takes HsmsMessage.
                    
                    # Let's try to access the connection via protocol object
                    if hasattr(equipment, 'protocol'):
                        print(f"Protocol attributes: {dir(equipment.protocol)}")
                        if hasattr(equipment.protocol, 'send_message'):
                            # HsmsProtocol usually has send_message
                            # But it might expect a HsmsMessage (which we have) or SecsMessage
                            # Let's try sending HsmsMessage
                            equipment.protocol.send_message(msg)
                            print("Sent S99F1 (via protocol.send_message)")
                        elif hasattr(equipment.protocol, 'connection') and equipment.protocol.connection:
                            equipment.protocol.connection.send_message(msg)
                            print("Sent S99F1 (via protocol.connection)")
                        else:
                            print("Could not find send_message or connection in protocol")
                    elif hasattr(equipment, 'connection') and equipment.connection:
                        equipment.connection.send_message(msg)
                        print("Sent S99F1")
                    else:
                        print("Could not find connection object to send raw message")
                    
                    time.sleep(1)
                    
                    # 2. Send Unknown Function (S1F99)
                    header2 = HsmsStreamFunctionHeader(
                        system=1002,
                        stream=1,
                        function=99,
                        session_id=10,
                        require_response=True
                    )
                    msg2 = HsmsMessage(header=header2, data=b'')
                    
                    if hasattr(equipment, 'protocol'):
                        if hasattr(equipment.protocol, 'send_message'):
                            equipment.protocol.send_message(msg2)
                            print("Sent S1F99 (via protocol.send_message)")
                        elif hasattr(equipment.protocol, 'connection') and equipment.protocol.connection:
                            equipment.protocol.connection.send_message(msg2)
                            print("Sent S1F99 (via protocol.connection)")
                    elif hasattr(equipment, 'connection') and equipment.connection:
                        equipment.connection.send_message(msg2)
                        print("Sent S1F99")

                except Exception as e:
                    print(f"Failed to send invalid messages: {e}")
                    import traceback
                    traceback.print_exc()

        # Check results
        print("\nVerification Results:")
        print(f"S9F3 (Unrecognized Stream) Received: {equipment.received_s9f3}")
        print(f"S9F5 (Unrecognized Function) Received: {equipment.received_s9f5}")
        
        if equipment.received_s9f3 and equipment.received_s9f5:
            print("SUCCESS: All S9 error tests passed!")
            sys.exit(0)
        else:
            print("FAILURE: Some tests failed.")
            sys.exit(1)

    except KeyboardInterrupt:
        equipment.disable()

if __name__ == "__main__":
    main()
