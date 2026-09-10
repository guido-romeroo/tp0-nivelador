package client

import (
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/logger"
)

type ClientConfig struct {
	AgencyId  uint16
	BatchSize uint16
}

func NewClientConfig(agencyId string, batchSize string) (*ClientConfig, error) {
	agencyIdUint, err := strconv.ParseUint(agencyId, 10, 16)
	if err != nil {
		return &ClientConfig{}, errors.New("AGENCY_ID environment variable must be a valid uint16")
	}

	batchSizeUint, err := strconv.ParseUint(batchSize, 10, 16)
	if err != nil {
		return &ClientConfig{}, errors.New("BATCH_SIZE environment variable must be a valid uint16")
	}

	if batchSizeUint == 0 {
		return &ClientConfig{}, errors.New("BATCH_SIZE environment variable must be greater than 0")
	}

	return &ClientConfig{
		AgencyId:  uint16(agencyIdUint),
		BatchSize: uint16(batchSizeUint),
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

	defer func() {
		if err := client.connectionWithNationalLottery.Close(); err != nil {
			logger.Error(
				"connection-close",
				"error", err,
			)
		}
	}()
	defer func() {
		if err := client.agencyRepository.Close(); err != nil {
			logger.Error(
				"agency-repository-close",
				"error", err,
			)
		}
	}()

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

		if !responseReceived.IsWinner() {
			return fmt.Errorf("expected WINNER, got %v", responseReceived.Type)
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

func (client *Client) sendBye() error {
	bye := newMessage(BYE, []byte{})
	if err := client.connectionWithNationalLottery.Send(bye); err != nil {
		logger.Error("sending-bye", logger.Fail)
		return err
	}
	return nil
}

func (client *Client) sendBetsToNationalLottery() error {
	maxSize := client.connectionWithNationalLottery.maxMessageSize
	betsPayload := make([]byte, 0, maxSize)
	betsInBatch := 0

	sendAndCleanPayload := func() error {
		if err := client.sendBets(betsPayload); err != nil {
			return err
		}
		betsPayload = betsPayload[:0]
		betsInBatch = 0
		return nil
	}

	for {
		bet, err := client.agencyRepository.NextBet(client.config.AgencyId)
		if err == io.EOF {
			err := client.sendBets(betsPayload)
			if err != nil {
				return err
			}
			err = client.sendBye()
			if err != nil {
				return err
			}
			break
		}
		if err != nil {
			return err
		}

		bytes := bet.ToBytes()

		if len(bytes) > maxSize {
			return errors.New("unexpected error: bet size exceeds maximum message size")
		}

		if len(bytes)+len(betsPayload) > maxSize || betsInBatch+1 > int(client.config.BatchSize) {
			if err := sendAndCleanPayload(); err != nil {
				return err
			}
		}
		betsPayload = append(betsPayload, bytes...)
		betsInBatch++
	}
	return nil
}

func (client *Client) sendBets(betsPayload []byte) error {
	if len(betsPayload) == 0 {
		return nil
	}
	message := newMessage(BETS, betsPayload)
	if err := client.connectionWithNationalLottery.Send(message); err != nil {
		logger.Error("sending-message", logger.Fail)
		return err
	}
	response, err := client.connectionWithNationalLottery.Recv()
	if err != nil {
		logger.Error("receiving-ack", logger.Fail)
		return err
	}
	if response.IsNack() {
		logger.Error("receiving-nack", logger.Fail)
		err := client.sendBye()
		if err != nil {
			return err
		}
		return errors.New("server rejected batch (NACK)")
	}

	if !response.IsAck() {
		return fmt.Errorf("expected ACK, got %v", response.Type)
	}
	return nil
}
