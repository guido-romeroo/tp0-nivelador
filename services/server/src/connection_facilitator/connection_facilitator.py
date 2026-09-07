import socket
import safe_socket
from protocol import Message

_HEADER_SIZE = 2
_MAX_MESSAGE_SIZE = 65535

class ConnectionFacilitator:
    def __init__(self, connection: socket.socket):
        self.header_size = _HEADER_SIZE
        self.max_message_size = _MAX_MESSAGE_SIZE
        self.connection = connection

    def close(self):
        if self.connection is not None:
            self.connection.close()
            self.connection = None

    def _check_message_size(self, message: bytes):
            message_size = len(message)

            if message_size > self.max_message_size:
                raise ValueError(
                    f"message size exceeds maximum allowed size "
                    f"of {self.max_message_size} bytes"
                )

    def _add_header_to_message(self, message: bytes) -> bytes:
        self._check_message_size(message)

        message_size = len(message)
        message_with_header = bytearray(
            self.header_size + message_size
        )

        message_with_header[0:2] = message_size.to_bytes(
            self.header_size,
            byteorder="big",
        )

        message_with_header[self.header_size:] = message

        return bytes(message_with_header)

    def send(self, msg: Message):
        if self.connection is None:
            raise RuntimeError("Connection is closed")

        data = msg.to_bytes()
        message_with_header = self._add_header_to_message(data)
        safe_socket.send_all(self.connection, message_with_header)

    def receive(self) -> Message:
        if self.connection is None:
            raise RuntimeError("Connection is closed")

        message_len_bytes = safe_socket.recv_all(
            self.connection,
            self.header_size,
        )

        message_len = int.from_bytes(
            message_len_bytes,
            byteorder="big",
        )

        msg_bytes = safe_socket.recv_all(
            self.connection,
            message_len,
        )

        return Message.from_bytes(msg_bytes)


    