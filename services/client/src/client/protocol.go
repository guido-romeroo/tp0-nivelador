package client

type MessageType byte

const (
	BET    MessageType = 0x01
	WINNER MessageType = 0x02
	BYE    MessageType = 0x03
)

type Message struct {
	Type    MessageType
	Payload []byte
}

func newMessage(messageType MessageType, payload []byte) *Message {
	return &Message{
		Type:    messageType,
		Payload: payload,
	}
}

func (m *Message) ToBytes() []byte {
	return append([]byte{byte(m.Type)}, m.Payload...)
}

func FromBytes(data []byte) *Message {
	return &Message{
		Type:    MessageType(data[0]),
		Payload: data[1:],
	}
}

func (m *Message) IsWinner() bool {
	return m.Type == WINNER
}

func (m *Message) IsBye() bool {
	return m.Type == BYE
}
