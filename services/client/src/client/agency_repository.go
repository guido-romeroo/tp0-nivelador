package client

import (
	"bufio"
	"errors"
	"io"
	"os"
	"strconv"
	"strings"
)

type AgencyConfig struct {
	BetsFilePath    string
	WinnersFilePath string
}

type AgencyRepository struct {
	betsFile    *os.File
	winnersFile *os.File
	bets        *bufio.Scanner
}

func NewAgencyRepository(config *AgencyConfig) (*AgencyRepository, error) {
	betsFile, err := os.Open(config.BetsFilePath)
	if err != nil {
		return nil, err
	}

	winnersFile, err := os.OpenFile(config.WinnersFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		errClose := betsFile.Close()
		return nil, errors.Join(err, errClose)
	}

	return &AgencyRepository{
		betsFile:    betsFile,
		winnersFile: winnersFile,
		bets:        bufio.NewScanner(betsFile),
	}, nil
}

func (repo *AgencyRepository) NextBet(agencyId uint16) (*Bet, error) {
	if !repo.bets.Scan() {
		if err := repo.bets.Err(); err != nil {
			return nil, err
		}
		return nil, io.EOF
	}
	record := strings.Split(repo.bets.Text(), ",")
	if len(record) != 5 {
		return nil, errors.New("invalid record format: expected 5 fields")
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

	return NewBet(
		agencyId,
		firstName,
		lastName,
		document,
		birthdate,
		number,
	)
}

func (repo *AgencyRepository) WriteWinner(bet *Bet) error {
	record := []string{
		bet.firstName,
		bet.lastName,
		strconv.FormatUint(uint64(bet.document), 10),
		bet.birthdate,
		strconv.FormatUint(uint64(bet.number), 10),
	}

	line := strings.Join(record, ",") + "\n"
	_, err := repo.winnersFile.WriteString(line)
	return err
}

func (repo *AgencyRepository) Close() error {
	errWinnersClose := repo.winnersFile.Close()
	errBetsClose := repo.betsFile.Close()

	return errors.Join(errWinnersClose, errBetsClose)
}
