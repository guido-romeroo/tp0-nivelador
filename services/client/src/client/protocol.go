package client

import (
	"encoding/binary"
	"errors"
)

type MessageType byte

const (
	BETS   MessageType = 0x01
	WINNER MessageType = 0x02
	ACK    MessageType = 0x03
	NACK   MessageType = 0x04
	BYE    MessageType = 0x05
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

func (m *Message) IsAck() bool {
	return m.Type == ACK
}

func (m *Message) IsNack() bool {
	return m.Type == NACK
}

func (m *Message) IsBye() bool {
	return m.Type == BYE
}

func (bet *Bet) ToBytes() []byte {
	size := 2 + 1 + len(bet.firstName) + 1 + len(bet.lastName) + 4 + 1 + len(bet.birthdate) + 4
	bytes := make([]byte, size)
	offset := 0
	binary.BigEndian.PutUint16(bytes[offset:offset+2], bet.agencyId)
	offset += 2

	offset += copy(bytes[offset:], []byte{byte(len(bet.firstName))})
	offset += copy(bytes[offset:], bet.firstName)

	offset += copy(bytes[offset:], []byte{byte(len(bet.lastName))})
	offset += copy(bytes[offset:], bet.lastName)

	binary.BigEndian.PutUint32(bytes[offset:offset+4], bet.document)
	offset += 4

	offset += copy(bytes[offset:], []byte{byte(len(bet.birthdate))})
	offset += copy(bytes[offset:], bet.birthdate)

	binary.BigEndian.PutUint32(bytes[offset:offset+4], bet.number)
	return bytes
}

func BetFromBytes(data []byte) (*Bet, error) {
	if len(data) < 2+2+2+4+2+4 {
		return nil, errors.New("data is too short to contain the minimum required fields for a Bet")
	}

	offset := 0
	agencyId := binary.BigEndian.Uint16(data[offset : offset+2])
	offset += 2

	firstNameLen := int(data[offset])
	offset++

	if len(data) < offset+firstNameLen {
		return nil, errors.New("data is too short to contain firstName")
	}

	firstName := string(data[offset : offset+firstNameLen])
	offset += firstNameLen

	if len(data) < offset+1 {
		return nil, errors.New("data is too short to contain lastName length")
	}
	lastNameLen := int(data[offset])
	offset++

	if len(data) < offset+lastNameLen {
		return nil, errors.New("data is too short to contain lastName")
	}
	lastName := string(data[offset : offset+lastNameLen])
	offset += lastNameLen

	if len(data) < offset+4 {
		return nil, errors.New("data is too short to contain document")
	}
	document := binary.BigEndian.Uint32(data[offset : offset+4])
	offset += 4

	if len(data) < offset+1 {
		return nil, errors.New("data is too short to contain birthdate length")
	}
	birthdateLen := int(data[offset])
	offset += 1

	if len(data) < offset+birthdateLen {
		return nil, errors.New("data is too short to contain birthdate")
	}
	birthdate := string(data[offset : offset+birthdateLen])
	offset += birthdateLen

	if len(data) < offset+4 {
		return nil, errors.New("data is too short to contain number")
	}
	number := binary.BigEndian.Uint32(data[offset : offset+4])

	return &Bet{
		agencyId:  agencyId,
		firstName: firstName,
		lastName:  lastName,
		document:  document,
		birthdate: birthdate,
		number:    number,
	}, nil
}
