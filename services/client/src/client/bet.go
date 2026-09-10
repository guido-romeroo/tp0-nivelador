package client

import (
	"errors"
)

type Bet struct {
	agencyId  uint16
	firstName string
	lastName  string
	document  uint32
	birthdate string
	number    uint32
}

func NewBet(agencyId uint16, firstName string, lastName string, document uint32, birthdate string, number uint32) (*Bet, error) {
	if len(firstName) > 255 {
		return nil, errors.New("first name exceeds maximum length of 255 characters")
	}
	if len(lastName) > 255 {
		return nil, errors.New("last name exceeds maximum length of 255 characters")
	}
	if len(birthdate) > 255 {
		return nil, errors.New("birthdate exceeds maximum length of 255 characters")
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
