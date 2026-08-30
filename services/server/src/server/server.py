import socket
import logger
import safe_socket
from lottery.bet import Bet

_ECHO_SERVER_MESSAGE_SIZE = 1024

def bet_from_bytes(data: bytes) -> Bet:
    offset = 0
    agency_id = int.from_bytes(data[offset:offset+2], byteorder="big")
    offset += 2

    first_name_length = int.from_bytes(data[offset], byteorder="big")
    offset += 1
    first_name = data[offset:offset+first_name_length].decode("utf-8")
    offset += first_name_length

    last_name_length = int.from_bytes(data[offset], byteorder="big")
    offset += 1
    last_name = data[offset:offset+last_name_length].decode("utf-8")
    offset += last_name_length

    document = int.from_bytes(data[offset:offset+4], byteorder="big")
    offset += 4

    birthdate_length = int.from_bytes(data[offset], byteorder="big")
    offset += 1
    birthdate = data[offset:offset+birthdate_length].decode("utf-8")
    offset += birthdate_length

    number = int.from_bytes(data[offset:offset+2], byteorder="big")

    return Bet(
        agency_id=agency_id,
        first_name=first_name,
        last_name=last_name,
        document=document,
        birthdate=birthdate,
        number=number
    )
class Server:
    def __init__(self, server_host: str, server_port: int) -> None:
        self.server_host = server_host
        self.server_port = server_port

    def _handle_client(self, client_socket):
        action = "handle-client"
        message_amount = 0
        try:
            logger.info(action, logger.LogResult.in_progress)
            while True:
                client_message = safe_socket.recv_all(
                    client_socket, _ECHO_SERVER_MESSAGE_SIZE
                )
                if not client_message:
                    logger.info(
                        action,
                        logger.LogResult.success,
                        "messages-amount",
                        message_amount,
                    )
                    return
                message_amount += 1
                safe_socket.send_all(client_socket, client_message)
        except Exception as e:
            logger.error(
                action, logger.LogResult.fail, "messages-amount", message_amount
            )
            raise e

    def run(self):
        action = "accept-connection"
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as server_socket:
            server_socket.bind((self.server_host, self.server_port))
            server_socket.listen()
            while True:
                try:
                    logger.info(action, logger.LogResult.in_progress)
                    client_socket, _ = server_socket.accept()
                except Exception as e:
                    logger.error(action, logger.LogResult.fail)
                    raise e
                logger.info(action, logger.LogResult.success)

                self._handle_client(client_socket)
