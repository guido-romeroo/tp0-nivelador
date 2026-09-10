package client

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/logger"
	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/safe_socket"
)

const CONNECTION_ATTEMPTS_MAX = 10
const CONNECTION_ATTEMPS_DELAY_MS = 200

type ConnectionConfig struct {
	Host string
	Port string
}

type ConnectionFacilitator struct {
	host           string
	port           string
	headerSize     int
	maxMessageSize int
	connection     net.Conn
}

// Se envía un mensaje con un header de 2 bytes que indica el tamaño del mensaje, seguido del mensaje en sí.
// Al hacer un recv, se lee primero el header de 2 bytes para saber el tamaño del mensaje, y luego se lee el mensaje completo.
func NewConnectionFacilitator(config *ConnectionConfig) (*ConnectionFacilitator, error) {
	connection, err := connectToServer(config.Host, config.Port)
	if err != nil {
		return nil, err
	}

	return &ConnectionFacilitator{
		host:           config.Host,
		port:           config.Port,
		headerSize:     2,
		maxMessageSize: 65535,
		connection:     connection,
	}, nil
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

func (connectionFacilitator *ConnectionFacilitator) Close() error {
	if connectionFacilitator.connection != nil {
		return connectionFacilitator.connection.Close()
	}
	return nil
}

func (connectionFacilitator *ConnectionFacilitator) checkMessageSize(message []byte) error {

	messageSize := int(len(message))
	if messageSize > connectionFacilitator.maxMessageSize {
		return fmt.Errorf("message size exceeds maximum allowed size of %d bytes", connectionFacilitator.maxMessageSize)
	}
	return nil
}

func (connectionFacilitator *ConnectionFacilitator) addHeaderToMessage(message []byte) ([]byte, error) {
	if err := connectionFacilitator.checkMessageSize(message); err != nil {
		return nil, err
	}
	messagewithHeader := make([]byte, connectionFacilitator.headerSize+len(message))

	messageSize := len(message)
	binary.BigEndian.PutUint16(messagewithHeader[0:2], uint16(messageSize))
	if copy(messagewithHeader[connectionFacilitator.headerSize:], message) != messageSize {
		return nil, errors.New("unexpected error while copying message to buffer with header")
	}
	return messagewithHeader, nil
}

func (connectionFacilitator *ConnectionFacilitator) Send(message *Message) error {
	messagewithHeader, err := connectionFacilitator.addHeaderToMessage(message.ToBytes())
	if err != nil {
		return err
	}
	return safe_socket.SendAll(connectionFacilitator.connection, messagewithHeader)
}

func (connectionFacilitator *ConnectionFacilitator) Recv() (*Message, error) {

	messageLen, err := safe_socket.RecvAll(connectionFacilitator.connection, connectionFacilitator.headerSize)
	if err != nil {
		return nil, err
	}

	message, err := safe_socket.RecvAll(connectionFacilitator.connection, int(binary.BigEndian.Uint16(messageLen)))
	if err != nil {
		return nil, err
	}
	return FromBytes(message), nil
}
