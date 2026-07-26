"""Self-contained HSMS linktest smoke test using the current secsgem API."""

import socket
import unittest

import secsgem.common
import secsgem.gem
import secsgem.hsms


def _free_local_port() -> int:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.bind(("127.0.0.1", 0))
        return int(sock.getsockname()[1])


class TestExampleSecsGem(unittest.TestCase):
    def setUp(self) -> None:
        port = _free_local_port()
        equipment_settings = secsgem.hsms.HsmsSettings(
            address="127.0.0.1",
            port=port,
            connect_mode=secsgem.hsms.HsmsConnectMode.PASSIVE,
            device_type=secsgem.common.DeviceType.EQUIPMENT,
            device_id=2,
        )
        host_settings = secsgem.hsms.HsmsSettings(
            address="127.0.0.1",
            port=port,
            connect_mode=secsgem.hsms.HsmsConnectMode.ACTIVE,
            device_type=secsgem.common.DeviceType.HOST,
            device_id=2,
        )

        self.equipment = secsgem.gem.GemEquipmentHandler(equipment_settings)
        self.host = secsgem.gem.GemHostHandler(host_settings)
        self.equipment.enable()
        self.host.enable()
        self.assertTrue(self.host.waitfor_communicating(5), "host did not reach COMMUNICATING")

    def tearDown(self) -> None:
        self.host.disable()
        self.equipment.disable()

    def test_linktest(self) -> None:
        response = self.host.protocol.send_linktest_req()

        self.assertIsNotNone(response)
        self.assertEqual(response.header.s_type.value, 6)
        self.assertEqual(response.header.device_id, 0xFFFF)


if __name__ == "__main__":
    unittest.main()
