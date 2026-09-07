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
			bye := newMessage(BYE, []byte{})
			if err := client.connectionWithNationalLottery.Send(bye); err != nil {
				return err
			}
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
	logger.Info("send-bet", logger.InProgress)
	messageToSend := newMessage(BET, bet.ToBytes())
	if err := client.connectionWithNationalLottery.Send(messageToSend); err != nil {
		logger.Error("send-bet", logger.Fail)
		return err
	}

	responseReceived, err := client.connectionWithNationalLottery.Recv()
	if err != nil {
		logger.Error("receive-winner", logger.Fail)
		return err
	}

	if len(responseReceived.Payload) != len(messageToSend.Payload) {
		err := fmt.Errorf("different lenghts between message and response, message-len: %d, response-len: %d", len(messageToSend.Payload), len(responseReceived.Payload))
		logger.Warn("different-lenghts", logger.Fail, "message", string(messageToSend.Payload), "response", string(responseReceived.Payload))
		return err
	}

	betOfServer, err := BetFromBytes(responseReceived.Payload)
	if err != nil {
		return err
	}

	if err := client.agencyRepository.WriteWinner(betOfServer); err != nil {
		return err
	}
	return nil
}
