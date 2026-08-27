package safe_socket

import "io"

//TODO: Complete with a short-read/short-write tolerant implementation

func SendAll(socket io.Writer, bytes []byte) error {
	for len(bytes) > 0 {
		n, err := socket.Write(bytes)
		if err != nil {
			return err
		}
		bytes = bytes[n:]
	}
	return nil
}

func RecvAll(socket io.Reader, size int) ([]byte, error) {
	buffer := make([]byte, size)
	received := 0
	for received < size {
		n, err := socket.Read(buffer[received:])
		if err != nil {
			return nil, err
		}
		received += n
	}
	return buffer, nil
}
