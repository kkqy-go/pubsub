package internal

// 此部分为内部频道，禁止外部模块使用
type EChannel int

const (
	EChannel_All EChannel = iota // 表示所有通道，用于订阅所有频道的消息
)
