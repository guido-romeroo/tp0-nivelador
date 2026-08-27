import socket

# TODO: Complete with a short-read/short-write tolerant implementation


def recv_all(socket: socket.socket, size):
    bytes = b""
    while len(bytes) < size:
        received = socket.recv(size - len(bytes))
        if received == b"":
            raise RuntimeError("socket connection broken")
        bytes += received
    return bytes
def send_all(socket: socket.socket, bytes):
    to_send = len(bytes)
    while to_send > 0:
        n = socket.send(bytes[len(bytes) - to_send:])
        if n == 0:
            raise RuntimeError("socket connection broken")
        to_send -= n

