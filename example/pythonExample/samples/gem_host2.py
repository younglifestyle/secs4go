from __future__ import annotations

import logging
import threading
import time

import secsgem.common
import secsgem.gem
import secsgem.hsms

from communication_log_file_handler import CommunicationLogFileHandler
from secsgem.gem.communication_state_machine import CommunicationState


def _value(node):
    try:
        return node.get()
    except Exception:
        return node


class GoEquipmentHost(secsgem.gem.GemHostHandler):
    def __init__(self, settings: secsgem.common.Settings) -> None:
        super().__init__(settings)
        self._event_cv = threading.Condition()
        self._last_event: dict | None = None

    # ---------- S6F11 handling ----------
    def _on_s06f11(self, handler, message):  # type: ignore[override]
        decoded = self.settings.streams_functions.decode(message)

        data_id = _value(getattr(decoded, "DATAID", None))
        ceid = _value(getattr(decoded, "CEID", None))

        reports: list[tuple[object, list[object]]] = []
        rpt_field = getattr(decoded, "RPT", None)
        if rpt_field is not None:
            try:
                entries = rpt_field.get()
            except Exception:
                entries = rpt_field
            for entry in entries or []:
                rptid = None
                values: list[object] = []
                if isinstance(entry, dict):
                    rptid = _value(entry.get("RPTID"))
                    v_field = entry.get("V")
                    if v_field is not None:
                        try:
                            v_list = v_field.get()
                        except Exception:
                            v_list = v_field
                        for item in v_list or []:
                            values.append(_value(item))
                else:
                    try:
                        items = entry.get()
                    except Exception:
                        items = entry
                    if items:
                        rptid = _value(items[0]) if len(items) > 0 else None
                        try:
                            for item in items[1].get():
                                values.append(_value(item))
                        except Exception:
                            pass
                reports.append((rptid, values))

        logging.info("S6F11 received: DATAID=%s CEID=%s REPORTS=%d", data_id, ceid, len(reports))

        with self._event_cv:
            self._last_event = {"DATAID": data_id, "CEID": ceid, "REPORTS": reports}
            self._event_cv.notify_all()

        return super()._on_s06f11(handler, message)

    def wait_event(self, timeout: float) -> dict | None:
        with self._event_cv:
            if self._last_event is not None:
                ev = self._last_event
                self._last_event = None
                return ev
            self._event_cv.wait(timeout)
            ev = self._last_event
            self._last_event = None
            return ev

    def _send(self, stream: int, function: int, body=None):
        # 1) 构造函数实例（把 body 放进去；有的 F 没 body，就传 None 或省略）
        fn = self.stream_function(stream, function)(body)

        # 2) 按需发送：大多数主机请求都是 W=1，需等待应答
        rsp = self.send_and_waitfor_response(fn)   # 返回 secsgem.common.Message 或 None
        if rsp is None:
            self.logger.error("S%02dF%02d no reply", stream, function)
            return None, None

        # 3) 解码
        decoded = self.settings.streams_functions.decode(rsp)
        try:
            payload = decoded.get()  # DataItem 容器
        except Exception:
            payload = decoded        # 已是原生结构

        self.logger.debug("S%02dF%02d decoded=%r payload=%r", stream, function, decoded, payload)
        return decoded, payload

    def _ack_value(self, decoded: object, payload: object, attr: str):
        if isinstance(decoded, dict) and attr in decoded:
            return _value(decoded[attr])
        node = getattr(decoded, attr, None)
        if node is not None:
            return _value(node)
        if isinstance(payload, dict) and attr in payload:
            return _value(payload[attr])
        return self._extract_binary(payload)

    @staticmethod
    def _extract_binary(payload: object):
        node = payload
        if isinstance(node, dict):
            return None
        if isinstance(node, (list, tuple)):
            if not node:
                return None
            node = node[0]
        return _value(node)

    # ---------- scenario ----------
    def demo_run(self) -> None:
        self._status_variable_namelist()
        self._status_snapshot()
        self._report_workflow()

        logging.info("waiting up to 8s for periodic S6F11 ...")
        if not self.wait_event(8.0):
            logging.warning("no event observed before remote command")

        self._remote_command()

    # ---------- S1F11/S1F12 ----------
    def _status_variable_namelist(self) -> None:
        logging.info("--- S1F11/S1F12 status variable namelist ---")
        # FIX: S1F11 expects a top-level Array (<L n> of <SVID>), not a dict.
        # Empty list [] means "all SVID names" on most tools per SEMI E5.
        decoded, payload = self._send(1, 11, [])
        entries = payload if isinstance(payload, (list, tuple)) else getattr(payload, "get", lambda: payload)()
        for entry in entries or []:
            if isinstance(entry, dict):
                svid = _value(entry.get("SVID"))
                name = _value(entry.get("SVNAME"))
                unit = _value(entry.get("UNITS"))
            else:
                try:
                    values = entry.get()
                except Exception:
                    values = entry
                svid = _value(values[0]) if values and len(values) > 0 else None
                name = _value(values[1]) if values and len(values) > 1 else ""
                unit = _value(values[2]) if values and len(values) > 2 else ""
            logging.info("SVID=%s NAME=%s UNIT=%s", svid, name, unit)

    # ---------- S1F3 snapshot ----------
    def _status_snapshot(self) -> None:
        logging.info("--- S1F3 status snapshot for SVID 1001 ---")
        decoded, payload = self._send(1, 3, [1001])
        values = payload if isinstance(payload, (list, tuple)) else getattr(payload, "get", lambda: payload)()
        if values:
            logging.info("SVID 1001 VALUE=%s", _value(values[0]))
        else:
            logging.warning("S1F3 returned empty list")

    # ---------- S2F33/S2F35/S2F37 ----------
    def _report_workflow(self) -> None:
        logging.info("--- report workflow ---")

        # S2F33 Clear: 正确键名是 DATA，不是 REPORTS
        try:
            self.stream_function(2, 33)({"DATAID": 0, "DATA": []})
        except Exception as exc:
            logging.debug("S2F33 clear ignored: %s", exc)

        # S2F33 Define Reports: DATA -> [{RPTID, VID}]
        define = {
            "DATAID": 0,
            "DATA": [
                {"RPTID": 4001, "VID": [1001, 2001]},
            ],
        }
        decoded, payload = self._send(2, 33, define)
        drack = self._ack_value(decoded, payload, "DRACK")
        if drack is not None:
            logging.info("S2F34 DRACK=%s", drack)
        else:
            logging.warning("S2F34 未返回DRACK, payload=%s", payload)

        self.report_subscriptions[4001] = [1001, 2001]

        # S2F35 Link Event Reports: DATA -> [{CEID, RPTID:[...]}]
        link = {
            "DATAID": 0,
            "DATA": [
                {"CEID": 3001, "RPTID": [4001]},
            ],
        }
        decoded, payload = self._send(2, 35, link)
        lrack = self._ack_value(decoded, payload, "LRACK")
        if lrack is not None:
            logging.info("S2F36 LRACK=%s", lrack)
        else:
            logging.warning("S2F36 未返回LRACK, payload=%s", payload)

        # S2F37 Enable/Disable Events: 字段名保持 CEED/CEID
        enable = {"CEED": True, "CEID": [3001]}
        decoded, payload = self._send(2, 37, enable)
        erack = self._ack_value(decoded, payload, "ERACK")
        if erack is not None:
            logging.info("S2F38 ERACK=%s", erack)
        else:
            logging.warning("S2F38 未返回ERACK, payload=%s", payload)

    # ---------- S2F41 remote command ----------
    def _remote_command(self) -> None:
        logging.info("--- remote command START (S2F41) ---")
        decoded, payload = self._send(2, 41, {"RCMD": "START", "PARAMS": []})
        hcack = self._ack_value(decoded, payload, "HCACK")
        logging.info("S2F42 HCACK=%s", hcack)

        ev = self.wait_event(6.0)
        if ev:
            logging.info("event after START: CEID=%s REPORTS=%d", ev.get("CEID"), len(ev.get("REPORTS", [])))
        else:
            logging.warning("no S6F11 observed after START")


def main() -> None:
    comm_handler = CommunicationLogFileHandler("log", "host2")
    comm_handler.setFormatter(logging.Formatter("%(asctime)s: %(message)s"))
    logging.getLogger("communication").addHandler(comm_handler)
    logging.getLogger("communication").propagate = False

    logging.basicConfig(
        format="%(asctime)s %(name)s.%(funcName)s: %(message)s",
        level=logging.INFO,
    )

    settings = secsgem.hsms.HsmsSettings(
        address="127.0.0.1",
        port=5000,
        connect_mode=secsgem.hsms.HsmsConnectMode.ACTIVE,
        device_type=secsgem.common.DeviceType.HOST,
        session_id=2,
    )

    host = GoEquipmentHost(settings)
    host.enable()

    try:
        deadline = time.time() + 10
        while host.communication_state.current_state.state != CommunicationState.COMMUNICATING:
            if time.time() >= deadline:
                logging.error("timeout waiting for COMMUNICATING state")
                break
            time.sleep(0.1)

        if host.communication_state.current_state.state == CommunicationState.COMMUNICATING:
            logging.info("Host in COMMUNICATING state")
            try:
                host.demo_run()
                logging.info("PY HOST2 RESULT: SUCCESS")
            except Exception:
                logging.exception("demo_run failed")
        else:
            logging.error("host never reached COMMUNICATING state")
    finally:
        host.disable()


if __name__ == "__main__":
    main()
