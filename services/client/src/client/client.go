package client

import (
	"io"

	"fmt"
	"strconv"

	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/logger"
)

type ClientConfig struct {
	AgencyId uint16
}

func NewClientConfig(agencyId string) (*ClientConfig, error) {
	agencyIdUint, err := strconv.ParseUint(agencyId, 10, 16)
	if err != nil {
		return &ClientConfig{}, fmt.Errorf("AGENCY_ID environment variable must be a valid uint16: %v", err)
	}
	return &ClientConfig{
		AgencyId: uint16(agencyIdUint),
	}, nil
}

type Client struct {
	connectionWithNationalLottery *ConnectionFacilitator
	agencyRepository              *AgencyRepository
	config                        *ClientConfig
}

func NewClient(config *ClientConfig, repository *AgencyRepository, connection *ConnectionFacilitator) *Client {
	return &Client{connectionWithNationalLottery: connection, agencyRepository: repository, config: config}

}

func (client *Client) Run() error {
	const mainAction = "client-run"

	defer client.connectionWithNationalLottery.Close()
	defer client.agencyRepository.Close()

	logger.Info(mainAction, logger.InProgress)

	err := client.sendBetsToNationalLottery()
	if err != nil {
		return err
	}

	err = client.writeWinnersFromNationalLottery()
	if err != nil {
		return err
	}

	logger.Info(mainAction, logger.Success, "agency-id", client.config.AgencyId)
	return nil
}

func (client *Client) writeWinnersFromNationalLottery() error {
	for {
		responseReceived, err := client.connectionWithNationalLottery.Recv()
		if err != nil {
			logger.Error("receiving-response", logger.Fail)
			return err
		}

		if responseReceived.IsBye() {
			break
		}

		winner, err := BetFromBytes(responseReceived.Payload)
		if err != nil {
			logger.Error("parsing-winner", logger.Fail)
			return err
		}

		if err := client.agencyRepository.WriteWinner(winner); err != nil {
			logger.Error("writing-winner", logger.Fail)
			return err
		}
	}
	return nil
}

func (client *Client) sendBetsToNationalLottery() error {
	for {

		bet, err := client.agencyRepository.NextBet(client.config.AgencyId)
		if err == io.EOF {
			bye := newMessage(BYE, []byte{})
			if err := client.connectionWithNationalLottery.Send(bye); err != nil {
				logger.Error("sending-bye", logger.Fail)
				return err
			}
			break
		}
		if err != nil {
			return err
		}

		message := newMessage(BET, bet.ToBytes())
		if err := client.connectionWithNationalLottery.Send(message); err != nil {
			logger.Error("sending-bet", logger.Fail)
			return err
		}
	}
	return nil
}
