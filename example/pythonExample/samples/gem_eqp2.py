import logging
import code
import threading
import time
import random

import secsgem.common
import secsgem.gem
import secsgem.hsms
import secsgem.secs

from communication_log_file_handler import CommunicationLogFileHandler


class EnhancedEquipment(secsgem.gem.GemEquipmentHandler):
    def __init__(self, settings: secsgem.common.Settings):
        super().__init__(settings)

        self.MDLN = "enhanced-equipment"
        self.SOFTREV = "1.1.0"

        # ---- runtime state ----
        self.temperature = 230
        self.pressure = 900
        self.run_id = self._new_run_id()

        # ---- 覆盖保留 SVID，避免 Clock(U4) 溢出 ----
        self.status_variables[1001] = secsgem.gem.StatusVariable(
            1001, "Clock", "", secsgem.secs.variables.String
        )
        self.status_variables[1002] = secsgem.gem.StatusVariable(
            1002, "ControlState", "", secsgem.secs.variables.U1
        )
        self.status_variables[1003] = secsgem.gem.StatusVariable(
            1003, "EventsEnabled", "", secsgem.secs.variables.U1
        )
        self.status_variables[1004] = secsgem.gem.StatusVariable(
            1004, "AlarmsEnabled", "", secsgem.secs.variables.U1
        )
        self.status_variables[1005] = secsgem.gem.StatusVariable(
            1005, "AlarmsSet", "", secsgem.secs.variables.U1
        )

        # ---- 自定义 SVID：用非保留号 1101 代表温度 ----
        self.status_variables.update(
            {
                1101: secsgem.gem.StatusVariable(
                    1101, "ChamberTemperature", "C", secsgem.secs.variables.U4
                ),
                2001: secsgem.gem.StatusVariable(
                    2001, "ChamberPressure", "Pa", secsgem.secs.variables.U4
                ),
                3002: secsgem.gem.StatusVariable(
                    3002, "CurrentRunId", "", secsgem.secs.variables.String
                ),
            }
        )

        # ---- 设备常量 ----
        self.ec1 = 321
        self.ec2 = "sample ec"
        self.equipment_constants.update(
            {
                20: secsgem.gem.EquipmentConstant(
                    20, "ChamberSetPoint", 0, 500, self.ec1, "degrees", secsgem.secs.variables.U4
                ),
                "EC2": secsgem.gem.EquipmentConstant(
                    "EC2", "RecipeName", 0, 0, 0, "chars", secsgem.secs.variables.String
                ),
            }
        )

        # ---- 事件 & 远程命令 ----
        self.collection_events.update({3001: secsgem.gem.CollectionEvent(3001, "LotStart", [])})
        self.remote_commands.update({
            "START": secsgem.gem.RemoteCommand("START", "Start Lot", [], 3001)
        })

        # ---- 简单配方存储 ----
        self._process_programs: dict[str, bytes] = {"SAMPLE": b"GDSCRIPT-001"}

    # ---------- helpers ----------
    @staticmethod
    def _new_run_id() -> str:
        return "RUN-" + time.strftime("%Y%m%d%H%M%S")

    def emit_event(self, ceid: int):
        # 多版本兜底：优先 data_collection，再 send_s6f11，最后 trigger_collection_events
        if hasattr(self, "data_collection") and hasattr(self.data_collection, "trigger_event"):
            return self.data_collection.trigger_event(ceid)
        if hasattr(self, "data_collection") and hasattr(self.data_collection, "trigger_collection_event"):
            return self.data_collection.trigger_collection_event(ceid)
        if hasattr(self, "send_s6f11"):
            return self.send_s6f11(ceid)
        if hasattr(self, "trigger_collection_events"):
            return self.trigger_collection_events([ceid])
        logging.warning("No known API to trigger CEID=%s in this secsgem version", ceid)

    def _delayed_emit_event(self, ceid: int, delay: float = 0.5):
        time.sleep(delay)
        self.emit_event(ceid)

    @staticmethod
    def _normalize_id(value):
        return value.get() if hasattr(value, "get") else value

    # ---------- callbacks ----------
    def on_sv_value_request(self, _svid, sv):
        if sv.svid == 1101:
            self.temperature = max(200, min(260, self.temperature + random.randint(-2, 2)))
            return sv.value_type(self.temperature)
        if sv.svid == 2001:
            self.pressure = max(880, min(940, self.pressure + random.randint(-3, 3)))
            return sv.value_type(self.pressure)
        if sv.svid == 3002:
            return sv.value_type(self.run_id)
        if sv.svid == 1001:  # Clock
            return sv.value_type(time.strftime("%Y%m%d%H%M%S"))
        return []

    def on_ec_value_request(self, ecid, ec):
        if ecid == 20:
            return ec.value_type(self.ec1)
        if ecid == "EC2":
            return ec.value_type(self.ec2)
        return []

    def on_ec_value_update(self, ecid, ec, value):
        if ecid == 20:
            self.ec1 = int(value)
        elif ecid == "EC2":
            self.ec2 = str(value)

    # ---------- S2F37 hook：Enable 后自动触发一次 ----------
    def _on_s02f37(self, handler, message):
        response = super()._on_s02f37(handler, message)
        try:
            erack = response.ERACK.get()
        except AttributeError:
            erack = None

        if erack == self.settings.data_items.ERACK.ACCEPTED:
            function = self.settings.streams_functions.decode(message)
            ceed = function.CEED.get()
            ceids = [self._normalize_id(ceid) for ceid in function.CEID.get()]
            if ceed and (not ceids or 3001 in ceids):
                threading.Thread(target=self._delayed_emit_event, args=(3001,), daemon=True).start()
        return response

    # ---------- Remote command handling ----------
    def _on_s02f41(self, handler, message):
        function = self.settings.streams_functions.decode(message)

        rcmd_name = function.RCMD.get()
        if isinstance(rcmd_name, str):
            rcmd_name = rcmd_name.strip()
        rcmd_callback_name = "rcmd_" + rcmd_name   # ✅ 这里拼的是 rcmd_START

        if rcmd_name not in self.remote_commands or not hasattr(self, rcmd_callback_name):
            logging.warning("remote command %s not available", rcmd_name)
            return self.stream_function(2, 42)({
                "HCACK": self.settings.data_items.HCACK.INVALID_COMMAND,
                "PARAMS": []
            })

        kwargs = {param["CPNAME"]: param["CPVAL"] for param in function.PARAMS.get()}

        try:
            getattr(self, rcmd_callback_name)(**kwargs)
        except Exception:
            logging.exception("remote command %s failed", rcmd_name)
            return self.stream_function(2, 42)({
                "HCACK": self.settings.data_items.HCACK.CANT_PERFORM_NOW,
                "PARAMS": []
            })

        return self.stream_function(2, 42)({
            "HCACK": self.settings.data_items.HCACK.ACK,
            "PARAMS": []
        })

    # ✅ 修正命名：与上面的拼接一致
    def rcmd_START(self, **_kwargs):
        logging.info("Executing Remote Command: START")
        self.run_id = self._new_run_id()
        self.emit_event(3001)

    # ---------- Process program upload/download ----------
    def _on_s07f03(self, handler, message):
        fn = self.settings.streams_functions.decode(message)
        ppid = fn.PPID.get()

        # 兼容字段名差异
        if hasattr(fn, "PPBODY"):
            body = fn.PPBODY.get()
        elif hasattr(fn, "PDATA"):
            body = fn.PDATA.get()
        else:
            body = b""

        if not isinstance(body, (bytes, bytearray)):
            body = str(body).encode("utf-8", errors="ignore")

        self._process_programs[str(ppid)] = body
        logging.info("Stored process program %s (%d bytes)", ppid, len(body))

        # ✅ 正确返回 ACKC7（字典形式）
        try:
            return self.stream_function(7, 4)({
                "ACKC7": self.settings.data_items.ACKC7.ACCEPTED
            })
        except Exception:
            # 少数版本只接受“裸枚举值”，做个兜底
            return self.stream_function(7, 4)(self.settings.data_items.ACKC7.ACCEPTED)

    def _on_s07f05(self, handler, message):
        fn = self.settings.streams_functions.decode(message)
        ppid = fn.get() if hasattr(fn, "get") else fn.PPID.get()

        body = self._process_programs.get(str(ppid), b"")
        if not isinstance(body, (bytes, bytearray)):
            body = str(body).encode("utf-8", errors="ignore")

        # ✅ 强制二进制类型（关键！）
        bin_body = secsgem.secs.variables.Binary(body)

        # 某些实现用 PPBODY，某些实现用 PDATA；按你的库两套都试一个优先
        try:
            return self.stream_function(7, 6)({"PPID": str(ppid), "PPBODY": bin_body})
        except Exception:
            # 兜底：改用 PDATA 键名
            return self.stream_function(7, 6)({"PPID": str(ppid), "PDATA": bin_body})


if __name__ == "__main__":
    commLogFileHandler = CommunicationLogFileHandler("log", "e")
    commLogFileHandler.setFormatter(logging.Formatter("%(asctime)s: %(message)s"))
    logging.getLogger("communication").addHandler(commLogFileHandler)
    logging.getLogger("communication").propagate = False

    logging.basicConfig(
        format="%(asctime)s %(name)s.%(funcName)s: %(message)s", level=logging.DEBUG
    )

    settings = secsgem.hsms.HsmsSettings(
        address="127.0.0.1",
        port=5000,
        connect_mode=secsgem.hsms.HsmsConnectMode.PASSIVE,
        device_type=secsgem.common.DeviceType.EQUIPMENT,
        session_id=2,
    )

    h = EnhancedEquipment(settings)
    h.enable()

    code.interact("equipment object is available as variable 'h'", local=locals())

    h.disable()
