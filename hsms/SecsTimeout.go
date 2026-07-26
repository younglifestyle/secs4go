package hsms

import "sync"

type SecsTimeout struct {
	mu       sync.RWMutex
	linktest int
	// default 45s, 回复超时 T3 (T3 reply timeout)表示一个实体等待回复消息的最长时间, 设备状态，则向主机发送 SECS-II S9F9 消息
	t3ReplyTimeout int
	// default 10s, 连接间隔时间 T5 (T5 connect separate timeout)表示两个连接请求之间的时间间隔
	t5ConnSeparateTimeout int
	// default 5s, 控制会话超时 T6 (T6 control transaction timeout)表示一个控制会话所能开启的最长时间，超过该时间就认为这次通信失败
	t6ControlTransTimeout int
	// default 10s, Not Select 状态超时 T7 (T7 NOT SELECT timeout)表示当建立了 TCP/IP 连接之后通信处于 Not Select 状态的最长时间，
	// 通信必须在该时间完成 select 操作，否则将会断开 TCP/IP 连接
	// 10 ~ 30 秒
	t7NotSelectTimeout int
	// default 6s, 网络字符超时 T8 (T8 network intercharacter timeout)表示成功接收到单个HSMS 消息的字符之间的最大时间间隔
	// 开始获取数据到结束的时间间隔
	// 45 ~ 65 秒最常见（测试时可能设 6~10 秒）
	t8NetworkIntercharTimeout int

	// Reconnection configuration
	// Maximum number of reconnection attempts (0 = unlimited)
	maxReconnectAttempts int
	// Base delay for exponential backoff (in seconds)
	reconnectBackoffBase int
	// Maximum delay for reconnection backoff (in seconds)
	reconnectBackoffMax int
	// Whether to enable automatic reconnection
	autoReconnect bool

	// Default device ID to use for S9 messages
	deviceID int
}

func (t *SecsTimeout) SetDeviceID(id int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.deviceID = id
}

func (t *SecsTimeout) DeviceID() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.deviceID
}

func NewSecsTimeout() *SecsTimeout {
	return &SecsTimeout{
		linktest:                  10, // 30
		t3ReplyTimeout:            45,
		t5ConnSeparateTimeout:     10,
		t6ControlTransTimeout:     5,
		t7NotSelectTimeout:        10,
		t8NetworkIntercharTimeout: 45,
		// Reconnection defaults
		maxReconnectAttempts: 10,
		reconnectBackoffBase: 2,
		reconnectBackoffMax:  60,
		autoReconnect:        true,
	}
}

func (s *SecsTimeout) SetLinktest(linktest int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.linktest = linktest
}

func (s *SecsTimeout) Linktest() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.linktest
}

func (s *SecsTimeout) T3ReplyTimeout() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.t3ReplyTimeout
}

func (s *SecsTimeout) SetT3ReplyTimeout(t3ReplyTimeout int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.t3ReplyTimeout = t3ReplyTimeout
}

func (s *SecsTimeout) T5ConnSeparateTimeout() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.t5ConnSeparateTimeout
}

func (s *SecsTimeout) SetT5ConnSeparateTimeout(t5ConnSeparateTimeout int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.t5ConnSeparateTimeout = t5ConnSeparateTimeout
}

func (s *SecsTimeout) T6ControlTransTimeout() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.t6ControlTransTimeout
}

func (s *SecsTimeout) SetT6ControlTransTimeout(t6ControlTransTimeout int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.t6ControlTransTimeout = t6ControlTransTimeout
}

func (s *SecsTimeout) T7NotSelectTimeout() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.t7NotSelectTimeout
}

func (s *SecsTimeout) SetT7NotSelectTimeout(t7NotSelectTimeout int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.t7NotSelectTimeout = t7NotSelectTimeout
}

func (s *SecsTimeout) T8NetworkIntercharTimeout() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.t8NetworkIntercharTimeout
}

func (s *SecsTimeout) SetT8NetworkIntercharTimeout(t8NetworkIntercharTimeout int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.t8NetworkIntercharTimeout = t8NetworkIntercharTimeout
}
