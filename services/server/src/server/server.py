import socket
import logger
from protocol import Message
from connection_facilitator import ConnectionFacilitator
from lottery.bet import Bet 
from lottery.lottery import Lottery

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

def bet_to_bytes(bet: Bet) -> bytes:
    first_name_bytes = bet.first_name.encode()
    last_name_bytes = bet.last_name.encode()
    birthdate_bytes = bet.birthdate.encode()

    return (
        bet.agency_id.to_bytes(2, byteorder="big") +
        len(first_name_bytes).to_bytes(1, byteorder="big") +
        first_name_bytes +
        len(last_name_bytes).to_bytes(1, byteorder="big") +
        last_name_bytes +
        bet.document.to_bytes(4, byteorder="big") +
        len(birthdate_bytes).to_bytes(1, byteorder="big") +
        birthdate_bytes +
        bet.number.to_bytes(2, byteorder="big")
    )

class Server:
    def __init__(self, server_host: str, server_port: int) -> None:
        self.server_host = server_host
        self.server_port = server_port

    def send_winner_bets_to_client(self, client_connection: ConnectionFacilitator, winner_bets: list[Bet]):
        for bet in winner_bets:
            bet_bytes = bet_to_bytes(bet)
            message = Message(Message.WINNER, bet_bytes)
            client_connection.send(message)
        bye = Message(Message.BYE, b'')
        client_connection.send(bye)


    def _handle_client(self, client_connection: ConnectionFacilitator):
        action = "handle-client"
        message_amount = 0
        agency_id = None
        lottery = Lottery("bets")
        try:
            logger.info(action, logger.LogResult.in_progress)
            while True:
                client_message = client_connection.receive()
                if client_message.is_bye():
                    logger.info(
                        action,
                        logger.LogResult.success,
                        "messages-amount",
                        message_amount,
                    )
                    break
                message_amount += 1
                bet = bet_from_bytes(client_message.payload)
                lottery.store_bets([bet])
                if agency_id is None:
                    agency_id = bet.agency_id
            winner_bets = [bet for bet in lottery.load_bets() if lottery.has_won(bet) and bet.agency_id == agency_id]
            self.send_winner_bets_to_client(client_connection, winner_bets)
        except Exception as e:
            logger.error(
                action, logger.LogResult.fail, "error", str(e)
            )
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
                    self._handle_client(client_connection)
                except Exception as e:
                    logger.error(action, logger.LogResult.fail)
                    self.close()
                    raise e
                logger.info(action, logger.LogResult.success)

