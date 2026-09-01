import socket
import logger
import safe_socket
from connection_facilitator import ConnectionFacilitator
from lottery.bet import Bet

def bet_from_bytes(data):
    min_size = 2 + 2 + 2 + 4 + 2 + 2

    if len(data) < min_size:
        raise ValueError("data is too short")

    offset = 0

    agency_id = int.from_bytes(data[offset:offset + 2], byteorder="big")
    offset += 2

    first_name_len = data[offset]
    offset += 1

    if len(data) < offset + first_name_len:
        raise ValueError("data is too short to contain firstName")

    first_name = data[offset:offset + first_name_len].decode()
    offset += first_name_len

    if len(data) < offset + 1:
        raise ValueError("data is too short to contain lastName length")

    last_name_len = data[offset]
    offset += 1

    if len(data) < offset + last_name_len:
        raise ValueError("data is too short to contain lastName")

    last_name = data[offset:offset + last_name_len].decode()
    offset += last_name_len

    if len(data) < offset + 4:
        raise ValueError("data is too short to contain document")

    document = int.from_bytes(data[offset:offset + 4], byteorder="big")
    offset += 4

    if len(data) < offset + 1:
        raise ValueError("data is too short to contain birthdate length")

    birthdate_len = data[offset]
    offset += 1

    if len(data) < offset + birthdate_len:
        raise ValueError("data is too short to contain birthdate")

    birthdate = data[offset:offset + birthdate_len].decode()
    offset += birthdate_len

    if len(data) < offset + 2:
        raise ValueError("data is too short to contain number")

    number = int.from_bytes(data[offset:offset + 2], byteorder="big")

    return Bet(
        agency_id=agency_id,
        first_name=first_name,
        last_name=last_name,
        document=document,
        birthdate=birthdate,
        number=number,
    )

class Server:
    def __init__(self, server_host: str, server_port: int) -> None:
        self.server_host = server_host
        self.server_port = server_port

    def _handle_client(self, client_connection: ConnectionFacilitator):
        action = "handle-client"
        message_amount = 0
        try:
            logger.info(action, logger.LogResult.in_progress)
            while True:
                client_message = client_connection.receive()
                if not client_message:
                    logger.info(
                        action,
                        logger.LogResult.success,
                        "messages-amount",
                        message_amount,
                    )
                    return
                message_amount += 1
                client_connection.send(client_message)
        except Exception as e:
            logger.error(
                action, logger.LogResult.fail, "messages-amount", message_amount
            )
            raise e
        finally:
            client_connection.close()

    def run(self):
        action = "accept-connection"
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as server_socket:
            server_socket.bind((self.server_host, self.server_port))
            server_socket.listen()
            while True:
                try:
                    logger.info(action, logger.LogResult.in_progress)
                    client_socket, _ = server_socket.accept()
                    client_connection = ConnectionFacilitator(client_socket)
                except Exception as e:
                    logger.error(action, logger.LogResult.fail)
                    self.close()
                    raise e
                logger.info(action, logger.LogResult.success)

                self._handle_client(client_connection)
