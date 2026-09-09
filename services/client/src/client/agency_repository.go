package client

import (
	"encoding/csv"
	"errors"
	"os"
	"strconv"
)

type AgencyConfig struct {
	BetsFilePath    string
	WinnersFilePath string
}

type AgencyRepository struct {
	bets        *csv.Reader
	winners     *csv.Writer
	betsFile    *os.File
	winnersFile *os.File
}

func NewAgencyRepository(config *AgencyConfig) (*AgencyRepository, error) {
	betsFile, err := os.Open(config.BetsFilePath)
	if err != nil {
		return nil, err
	}

	winnersFile, err := os.OpenFile(config.WinnersFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		betsFile.Close()
		return nil, err
	}

	return &AgencyRepository{
		betsFile:    betsFile,
		winnersFile: winnersFile,
		bets:        csv.NewReader(betsFile),
		winners:     csv.NewWriter(winnersFile),
	}, nil
}

func (repo *AgencyRepository) NextBet(agencyId uint16) (*Bet, error) {
	record, err := repo.bets.Read()
	if err != nil {
		return nil, err
	}

	firstName := record[0]
	lastName := record[1]

	val, err := strconv.ParseUint(record[2], 10, 32)
	if err != nil {
		return nil, err
	}
	document := uint32(val)
	birthdate := record[3]

	val, err = strconv.ParseUint(record[4], 10, 32)
	if err != nil {
		return nil, err
	}
	number := uint32(val)

	bet, err := NewBet(agencyId, firstName, lastName, document, birthdate, number)

	return bet, nil
}

func (repo *AgencyRepository) WriteWinner(bet *Bet) error {
	record := []string{
		bet.firstName,
		bet.lastName,
		strconv.FormatUint(uint64(bet.document), 10),
		bet.birthdate,
		strconv.FormatUint(uint64(bet.number), 10),
	}

	if err := repo.winners.Write(record); err != nil {
		return err
	}

	repo.winners.Flush()

	return repo.winners.Error()
}

func (repo *AgencyRepository) Close() error {
	repo.winners.Flush()

	errFlush := repo.winners.Error()

	errWinnersClose := repo.winnersFile.Close()

	errBetsClose := repo.betsFile.Close()

	return errors.Join(errFlush, errWinnersClose, errBetsClose)
}
