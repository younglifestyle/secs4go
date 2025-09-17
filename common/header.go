package common

const (
	STypeDataMessage = "data message"
	STypeSelectReq   = "select.req"
	STypeSelectRsp   = "select.rsp"
	STypeDeselectReq = "deselect.req"
	STypeDeselectRsp = "deselect.rsp"
	STypeLinktestReq = "linktest.req"
	STypeLinktestRsp = "linktest.rsp"
	STypeRejectReq   = "reject.req"
	STypeSeparateReq = "separate.req"
)

var (
	SType = map[string]int{
		"data message": 0, "select.req": 1, "select.rsp": 2, "deselect.req": 3, "deselect.rsp": 4,
		"linktest.req": 5, "linktest.rsp": 6, "reject.req": 7, "separate.req": 9,
	}
)
