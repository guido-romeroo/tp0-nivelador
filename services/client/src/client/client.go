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

func openFileAndLogErrorIfAny(filePath string, flag int, permission os.FileMode) (*os.File, error) {
	file, err := os.OpenFile(filePath, flag, permission)
	if err != nil {
		logger.Warn("open-file", logger.Fail, "file", filePath)
		return nil, err
	}
	return file, nil
}

func (client *Client) Run() error {
	const mainAction = "test-echo-server"
	defer client.conn.Close()
	logger.Info(mainAction, logger.InProgress)

	fileInput, err := openFileAndLogErrorIfAny(client.config.InputFile, os.O_RDONLY, 0644)
	if err != nil {
		return err
	}
	defer fileInput.Close()

	fileOutput, err := openFileAndLogErrorIfAny(client.config.OutputFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer fileOutput.Close()

	scanner := bufio.NewScanner(fileInput)

	for scanner.Scan() {
		messageToSend := scanner.Bytes()
		err1 := client.sendMessageFromFileInputAndWriteEchoInFileOutput(messageToSend, fileOutput)
		if err1 != nil {
			return err1
		}
	}

	if err := scanner.Err(); err != nil {
		logger.Warn("scanner-err", logger.Fail)
		return err
	}

	logger.Info(mainAction, logger.Success, "agency-id", client.config.AgencyId)
	return nil
}

func (client *Client) sendMessageFromFileInputAndWriteEchoInFileOutput(messageToSend []byte, fileOutput *os.File) error {
	if err := safe_socket.SendAll(client.conn, messageToSend); err != nil {
		logger.Warn("send-message", logger.Fail, "message", string(messageToSend))
		return err
	}

	responseReceived, err := safe_socket.RecvAll(client.conn, ECHO_CLIENT_BUFFER_SIZE)
	if err != nil {
		logger.Warn("recv-response", logger.Fail, "message", string(messageToSend))
		return err
	}

	if len(responseReceived) != len(messageToSend) {
		err := fmt.Errorf("different lenghts between message and response, message-len: %d, response-len: %d", len(messageToSend), len(responseReceived))
		logger.Warn("different-lenghts", logger.Fail, "message", string(messageToSend), "response", string(responseReceived))
		return err
	}

	line := append(responseReceived, '\n')
	n_bytes, err := fileOutput.Write(line)
	if err != nil {
		logger.Warn("write-file", logger.Fail)
		return err
	}

	if n_bytes != len(line) {
		logger.Warn("write-file", logger.Fail)
		return fmt.Errorf("different lenghts between response and bytes-writed, len msg: %d, len bytes-writed: %d", len(line), n_bytes)
	}
	return nil
}
