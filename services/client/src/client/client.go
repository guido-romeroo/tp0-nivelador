package client

import (
	"io"

	"fmt"
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
		return &ClientConfig{}, fmt.Errorf("AGENCY_ID environment variable must be a valid uint16: %v", err)
	}

	batchSizeUint, err := strconv.ParseUint(batchSize, 10, 16)
	if err != nil {
		return &ClientConfig{}, fmt.Errorf("BATCH_SIZE environment variable must be a valid uint16: %v", err)
	}

	if batchSizeUint == 0 {
		return &ClientConfig{}, fmt.Errorf("BATCH_SIZE environment variable must be greater than 0")
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
	maxSize := client.connectionWithNationalLottery.maxMessageSize
	batchPayload := make([]byte, 0, maxSize)
	betsInBatch := 0

	sendAndCleanPayload := func() error {
		if err := client.sendBatch(batchPayload); err != nil {
			return err
		}
		batchPayload = batchPayload[:0]
		betsInBatch = 0
		return nil
	}

	for {
		bet, err := client.agencyRepository.NextBet(client.config.AgencyId)
		if err == io.EOF {
			err := client.sendBatch(batchPayload)
			if err != nil {
				return err
			}
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

		bytes := bet.ToBytes()

		if len(bytes) > maxSize {
			return fmt.Errorf("unexpected error: bet size exceeds maximum message size. Bet: %v, MaxSize: %d", bet, maxSize)
		}

		if len(bytes)+len(batchPayload) > maxSize || betsInBatch+1 > int(client.config.BatchSize) {
			if err := sendAndCleanPayload(); err != nil {
				return err
			}
		}
		batchPayload = append(batchPayload, bytes...)
		betsInBatch++
	}
	return nil
}

func (client *Client) sendBatch(batchPayload []byte) error {
	if len(batchPayload) == 0 {
		return nil
	}
	batchMessage := newMessage(BATCH, batchPayload)
	if err := client.connectionWithNationalLottery.Send(batchMessage); err != nil {
		logger.Error("sending-batch", logger.Fail)
		return err
	}
	response, err := client.connectionWithNationalLottery.Recv()
	if err != nil {
		logger.Error("receiving-ack", logger.Fail)
		return err
	}
	if response.IsNack() {
		logger.Error("receiving-nack", logger.Fail)
		return fmt.Errorf("server rejected batch (NACK)")
	}

	if !response.IsAck() {
		return fmt.Errorf("expected ACK, got %v", response.Type)
	}
	return nil
}
