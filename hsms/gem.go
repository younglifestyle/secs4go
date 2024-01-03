package hsms

//type Gem struct {
//	session     *link.Session
//	DeviceID    uint16 // Session ID
//	MDLN        string
//	SOFTREV     string
//	MessageBuff map[uint32]struct{}
//	sync.Mutex
//}

//func s9f9(session *link.Session) error {
//	state := session.State.(*SessionState)
//
//	return session.Send(ast.NewDataMessage("TransactionTimeout", 9, 9,
//		0, "H<-E", ast.NewEmptyItemNode()).
//		SetSessionIDAndSystemBytes(
//			int(state.secsEQPConfig.DeviceID), genSystemByte(state.SystemID)).ToBytes())
//}
//
//func s1f14(session *link.Session) error {
//	state := session.State.(*SessionState)
//	return session.Send(ast.NewDataMessage("EstablishAck", 1, 14,
//		0, "H<-E",
//		ast.NewListNode(
//			ast.NewBinaryNode(0),
//			ast.NewListNode(ast.NewASCIINode(state.secsEQPConfig.MDLN), ast.NewASCIINode(state.secsEQPConfig.SOFTREV)))).
//		SetSessionIDAndSystemBytes(
//			int(state.RemoteSessionID), genSystemByte(state.SystemID)).ToBytes())
//}
