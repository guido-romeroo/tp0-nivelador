import socket
import logger
from protocol import Message, bet_to_bytes, bets_from_bytes
from connection_facilitator import ConnectionFacilitator
from lottery import Lottery
from lottery import Bet

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
                if not client_message.is_batch() and  not client_message.is_bet():
                    raise ValueError(f"invalid message type: {client_message.type}")
                bets = bets_from_bytes(client_message.payload)
                if agency_id is None and bets:
                    agency_id = bets[0].agency_id
                    
                try:
                    lottery.store_bets(bets)
                except Exception as e:
                    logger.error(
                        action,
                        logger.LogResult.fail,
                        "error while storing bets, sending NACK, error",
                        str(e)
                    )
                    nacked_message = Message(Message.NACK, b'')
                    client_connection.send(nacked_message)
                    continue
                acked_message = Message(Message.ACK, b'')
                client_connection.send(acked_message)
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
                except Exception as e:
                    logger.error(action, logger.LogResult.fail)
                    raise e
                logger.info(action, logger.LogResult.success)
                client_connection = ConnectionFacilitator(client_socket)
                self._handle_client(client_connection)

