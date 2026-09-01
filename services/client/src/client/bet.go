package client

import (
	"encoding/binary"
	"fmt"
)

type Bet struct {
	agencyId  uint16
	firstName string
	lastName  string
	document  uint32
	birthdate string
	number    uint16
}

func NewBet(agencyId uint16, firstName string, lastName string, document uint32, birthdate string, number uint16) (*Bet, error) {
	if len(firstName) > 255 {
		return nil, fmt.Errorf("first name exceeds maximum length of 255 characters")
	}
	if len(lastName) > 255 {
		return nil, fmt.Errorf("last name exceeds maximum length of 255 characters")
	}
	if len(birthdate) > 255 {
		return nil, fmt.Errorf("birthdate exceeds maximum length of 255 characters")
	}
	return &Bet{
		agencyId:  agencyId,
		firstName: firstName,
		lastName:  lastName,
		document:  document,
		birthdate: birthdate,
		number:    number,
	}, nil
}

func (bet *Bet) ToBytes() []byte {
	size := 2 + 1 + len(bet.firstName) + 1 + len(bet.lastName) + 4 + 1 + len(bet.birthdate) + 2
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

	binary.BigEndian.PutUint16(bytes[offset:offset+2], bet.number)
	return bytes
}

func BetFromBytes(data []byte) (*Bet, error) {
	if len(data) < 2+2+2+4+2+2 {
		return nil, fmt.Errorf("data is too short to contain the minimum required fields for a Bet")
	}

	offset := 0
	agencyId := binary.BigEndian.Uint16(data[offset : offset+2])
	offset += 2

	firstNameLen := int(data[offset])
	offset++

	if len(data) < offset+firstNameLen {
		return nil, fmt.Errorf("data is too short to contain firstName")
	}

	firstName := string(data[offset : offset+firstNameLen])
	offset += firstNameLen

	if len(data) < offset+1 {
		return nil, fmt.Errorf("data is too short to contain lastName length")
	}
	lastNameLen := int(data[offset])
	offset++

	if len(data) < offset+lastNameLen {
		return nil, fmt.Errorf("data is too short to contain lastName")
	}
	lastName := string(data[offset : offset+lastNameLen])
	offset += lastNameLen

	if len(data) < offset+4 {
		return nil, fmt.Errorf("data is too short to contain document")
	}
	document := binary.BigEndian.Uint32(data[offset : offset+4])
	offset += 4

	if len(data) < offset+1 {
		return nil, fmt.Errorf("data is too short to contain birthdate length")
	}
	birthdateLen := int(data[offset])
	offset += 1

	if len(data) < offset+birthdateLen {
		return nil, fmt.Errorf("data is too short to contain birthdate")
	}
	birthdate := string(data[offset : offset+birthdateLen])
	offset += birthdateLen

	if len(data) < offset+2 {
		return nil, fmt.Errorf("data is too short to contain number")
	}
	number := binary.BigEndian.Uint16(data[offset : offset+2])

	return &Bet{
		agencyId:  agencyId,
		firstName: firstName,
		lastName:  lastName,
		document:  document,
		birthdate: birthdate,
		number:    number,
	}, nil
}
