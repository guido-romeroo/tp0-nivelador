package client

import (
	"net"
	"time"

	"bufio"
	"fmt"
	"os"

	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/logger"
	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/safe_socket"
)

const CONNECTION_ATTEMPTS_MAX = 3
const CONNECTION_ATTEMPS_DELAY_MS = 200

const ECHO_CLIENT_BUFFER_SIZE = 512
const ECHO_CLIENT_MESSAGE_AMOUNT = 3
const ECHO_CLIENT_MESSAGE_DELAY_MS = 1000

type ClientConfig struct {
	ServerHost string
	ServerPort string
	AgencyId   string
	InputFile  string
	OutputFile string
}

type Client struct {
	conn   net.Conn
	config ClientConfig
}

func NewClient(config ClientConfig) (*Client, error) {
	conn, err := connectToServer(config.ServerHost, config.ServerPort)
	if err != nil {
		logger.Warn("connect-to-server", logger.Fail)
		return nil, err
	}

	client := &Client{conn: conn, config: config}
	return client, nil
}

func connectToServer(host, port string) (net.Conn, error) {
	const action = "connect-to-server"
	var err error
	var conn net.Conn

	logger.Info(action, logger.InProgress)
	for i := range CONNECTION_ATTEMPTS_MAX {
		conn, err = net.Dial("tcp", host+":"+port)
		if err != nil {
			logger.Warn(action, logger.Fail, "attempt", i)
			time.Sleep(CONNECTION_ATTEMPS_DELAY_MS * time.Millisecond)
			continue
		}

		logger.Info(action, logger.Success)
		break
	}

	return conn, err
}

func (client *Client) Run() error {
	const mainAction = "test-echo-server"
	defer client.conn.Close()

	fileInput, err := os.OpenFile(client.config.InputFile, os.O_RDONLY, 0644)
	if err != nil {
		logger.Error("open-file", logger.Fail, "file", client.config.InputFile)
		return err
	}
	defer fileInput.Close()

	fileOutput, err := os.OpenFile(client.config.OutputFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		logger.Error("open-file", logger.Fail, "file", client.config.OutputFile)
		return err
	}
	defer fileOutput.Close()

	scanner := bufio.NewScanner(fileInput)

	for scanner.Scan() {
		lineBytes := scanner.Bytes()
		if err := safe_socket.SendAll(client.conn, lineBytes); err != nil {
			logger.Error("send-message", logger.Fail)
			return err
		}

		responseBuffer, err := safe_socket.RecvAll(client.conn, ECHO_CLIENT_BUFFER_SIZE)
		if err != nil {
			logger.Error("recv-response", logger.Fail)
			return err
		}

		if len(responseBuffer) != len(lineBytes) {
			fileOutput.Write(lineBytes)
			fileOutput.Write([]byte{'\n'})
			fmt.Fprintf(fileOutput, "%d", len(lineBytes))
			fileOutput.Write([]byte{'\n'})

			fileOutput.Write(responseBuffer)
			fileOutput.Write([]byte{'\n'})
			fmt.Fprintf(fileOutput, "%d", len(responseBuffer))

			logger.Error("different lenghts between request-response", logger.LogResult(responseBuffer), logger.LogResult(lineBytes), logger.Fail)
			return fmt.Errorf("unexpected response, len of request must be equal to the response")
		}

		line := append(responseBuffer, '\n')
		n_bytes, err := fileOutput.Write(line)
		if err != nil {
			logger.Error("write-file", logger.Fail)
			return err
		}

		if n_bytes != len(line) {
			logger.Error("different lenghts between response and bytes-writed", logger.Fail)
			return fmt.Errorf("unexpected error while writing in output file, len of bytes writed must be equal to the response len")
		}
	}

	if err := scanner.Err(); err != nil {
		logger.Error("scanner-err", logger.Fail)
		return err
	}

	logger.Info(mainAction, logger.Success, "agency-id", client.config.AgencyId)
	return nil
}
