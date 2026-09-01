package main

import (
	"errors"
	"os"

	client "github.com/7574-sistemas-distribuidos/tp-nivelador/src/client"
	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/logger"
)

func loadAgencyConfig() (*client.AgencyConfig, error) {
	inputFile := os.Getenv("INPUT_FILE")

	if inputFile == "" {
		return &client.AgencyConfig{}, errors.New("INPUT_FILE environment variable is required")
	}

	outputFile := os.Getenv("OUTPUT_FILE")

	if outputFile == "" {
		return &client.AgencyConfig{}, errors.New("OUTPUT_FILE environment variable is required")
	}

	return &client.AgencyConfig{
		BetsFilePath:    inputFile,
		WinnersFilePath: outputFile,
	}, nil
}

func loadClientConfig() (*client.ClientConfig, error) {
	agencyId := os.Getenv("AGENCY_ID")
	if agencyId == "" {
		return &client.ClientConfig{}, errors.New("AGENCY_ID environment variable is required")
	}

	return client.NewClientConfig(agencyId)
}

func loadConnectionConfig() (*client.ConnectionConfig, error) {
	serverHost := os.Getenv("SERVER_HOST")
	if serverHost == "" {
		return &client.ConnectionConfig{}, errors.New("SERVER_HOST environment variable is required")
	}

	serverPort := os.Getenv("SERVER_PORT")
	if serverPort == "" {
		return &client.ConnectionConfig{}, errors.New("SERVER_PORT environment variable is required")
	}

	return &client.ConnectionConfig{
		Host: serverHost,
		Port: serverPort,
	}, nil
}

func loadConfig() (*client.ClientConfig, *client.AgencyConfig, *client.ConnectionConfig, error) {
	clientConfig, err := loadClientConfig()
	if err != nil {
		return &client.ClientConfig{}, &client.AgencyConfig{}, &client.ConnectionConfig{}, err
	}

	agencyConfig, err := loadAgencyConfig()
	if err != nil {
		return &client.ClientConfig{}, &client.AgencyConfig{}, &client.ConnectionConfig{}, err
	}

	connectionConfig, err := loadConnectionConfig()
	if err != nil {
		return &client.ClientConfig{}, &client.AgencyConfig{}, &client.ConnectionConfig{}, err
	}

	return clientConfig, agencyConfig, connectionConfig, nil
}

func run() int {
	client_config, agency_config, connection_config, err := loadConfig()
	if err != nil {
		logger.Error("load-config", logger.Fail, "err", err)
		return 1
	}

	connection, err := client.NewConnectionFacilitator(connection_config)
	if err != nil {
		logger.Error("connection-facilitator", logger.Fail, "err", err)
		return 1
	}

	repository, err := client.NewAgencyRepository(agency_config)
	if err != nil {
		logger.Error("agency-repository", logger.Fail, "err", err)
		return 1
	}

	client := client.NewClient(client_config, repository, connection)

	if err := client.Run(); err != nil {
		logger.Error("client-run", logger.Fail, "err", err)
		return 1
	}
	return 0
}

func main() {
	os.Exit(run())
}
