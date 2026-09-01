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
	const mainAction = "test-echo-server"

	defer client.connectionWithNationalLottery.Close()
	defer client.agencyRepository.Close()

	logger.Info(mainAction, logger.InProgress)

	for {

		bet, err := client.agencyRepository.NextBet(client.config.AgencyId)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if err := client.sendBetsAndReceiveWinners(bet); err != nil {
			return err
		}
	}

	logger.Info(mainAction, logger.Success, "agency-id", client.config.AgencyId)
	return nil
}

func (client *Client) sendBetsAndReceiveWinners(bet *Bet) error {
	messageToSend := bet.ToBytes()
	if err := client.connectionWithNationalLottery.Send(messageToSend); err != nil {
		return err
	}

	responseReceived, err := client.connectionWithNationalLottery.Recv()
	if err != nil {
		return err
	}

	if len(responseReceived) != len(messageToSend) {
		err := fmt.Errorf("different lenghts between message and response, message-len: %d, response-len: %d", len(messageToSend), len(responseReceived))
		logger.Warn("different-lenghts", logger.Fail, "message", string(messageToSend), "response", string(responseReceived))
		return err
	}

	betOfServer, err := BetFromBytes(responseReceived)
	if err != nil {
		return err
	}

	if err := client.agencyRepository.WriteWinner(betOfServer); err != nil {
		return err
	}
	return nil
}
