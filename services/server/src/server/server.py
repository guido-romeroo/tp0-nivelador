import socket
import threading
import queue
import logger
from protocol import Message, bet_to_bytes, bets_from_bytes
from connection_facilitator import ConnectionFacilitator
from lottery import Lottery
from lottery import Bet

class CoordinateMessage:
    BETS = 1
    START_RAFFLE = 2
    FINISHED = 3

    def __init__(self, message_type: int, content: tuple[list[Bet], queue.Queue] | queue.Queue | threading.Thread):
        self.message_type = message_type
        self.content = content

    def is_bets(self):
        return self.message_type == CoordinateMessage.BETS
    def is_start_raffle(self):
        return self.message_type == CoordinateMessage.START_RAFFLE

    def is_finished(self):
        return self.message_type == CoordinateMessage.FINISHED

class Server:
    def __init__(self, server_host: str, server_port: int, agency_quorum_min: int) -> None:
        self.server_host = server_host
        self.server_port = server_port
        self.agency_quorum_min = agency_quorum_min

    def send_winner_bets_to_client(self, client_connection: ConnectionFacilitator, winner_bets: list[Bet]):
        for bet in winner_bets:
            message = Message(Message.WINNER, bet_to_bytes(bet))
            client_connection.send(message)
        bye = Message(Message.BYE, b'')
        client_connection.send(bye)

    def receive_bets_from_client(self, client_connection: ConnectionFacilitator) -> list[Bet]:
        action = "receive-bets-from-client"
        client_message = client_connection.receive()
        if client_message.is_bye():
            logger.info(
                action,
                logger.LogResult.success,
            )
            return []
        if not client_message.is_batch() and not client_message.is_bet():
            err = f"invalid message type: {client_message.type}"
            logger.error(action, logger.LogResult.fail, "error", err)
            raise ValueError(err)
        return bets_from_bytes(client_message.payload)

    def send_bets_to_coordinator(self, coordinator_channel: queue.Queue, bets: list[Bet]):
        ack_channel = queue.Queue()
        coordinator_channel.put(CoordinateMessage(CoordinateMessage.BETS, (bets, ack_channel)))
        if not ack_channel.get():
            logger.error(
                "send-bets-to-coordinator",
                logger.LogResult.fail,
                "coordinator failed to store bets, sending NACK to client"
            )
            return False
        return True

    def receive_winners_from_coordinator(self, coordinator_channel: queue.Queue, agency_id: int) -> list[Bet]:
        winners_channel = queue.Queue()
        coordinator_channel.put(CoordinateMessage(CoordinateMessage.START_RAFFLE, winners_channel))
        winner_bets = winners_channel.get()
        selected = [bet for bet in winner_bets if bet.agency_id == agency_id]
        return selected
            
    def _handle_client(self, client_connection: ConnectionFacilitator, coordinator_channel: queue.Queue):
        agency_id = None
        try:
            action = "handle-client"
            logger.info(action, logger.LogResult.in_progress)
            while True:
                bets = self.receive_bets_from_client(client_connection)
                if not bets:
                    break
                if agency_id is None:
                    agency_id = bets[0].agency_id

                if not self.send_bets_to_coordinator(coordinator_channel, bets):
                    client_connection.send(Message(Message.NACK, b''))
                    continue
                client_connection.send(Message(Message.ACK, b''))

            winner_bets = self.receive_winners_from_coordinator(coordinator_channel, agency_id)
            self.send_winner_bets_to_client(client_connection, winner_bets)
        except Exception as e:
            logger.error(
                action, logger.LogResult.fail, "error", str(e)
            )
        finally:
            coordinator_channel.put(CoordinateMessage(CoordinateMessage.FINISHED, threading.current_thread()))
            client_connection.close()

    def accept_connection(self, server_socket: socket.socket):
        action = "accept-connection"
        try:
            logger.info(action, logger.LogResult.in_progress)
            client_socket, _ = server_socket.accept()
        except Exception as e:
            logger.error(action, logger.LogResult.fail)
            raise e
        logger.info(action, logger.LogResult.success)
        return ConnectionFacilitator(client_socket)

    def accept_connections(self, coordinator_channel: queue.Queue):
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as server_socket:
            server_socket.bind((self.server_host, self.server_port))
            server_socket.listen()
            while True:
                connection = self.accept_connection(server_socket)
                thread = threading.Thread(target=self._handle_client, args=(connection, coordinator_channel))
                thread.start() #coordinator hace join de los threads que se comunican con el cliente


    def store_bets(self, lottery: Lottery, content: tuple[list[Bet], queue.Queue]):
        bets = content[0]
        ack_channel = content[1]
        try:
            lottery.store_bets(bets)
        except Exception as e:
            logger.error(
                "store-bets",
                logger.LogResult.fail,
                "error while storing bets",
                str(e),
                "bets",
                str(bets)
            )
            ack_channel.put(False)
            return
        ack_channel.put(True)

    def send_winners(self, lottery: Lottery, client_thread_channels: list[queue.Queue]):
        winner_bets = [bet for bet in lottery.load_bets() if lottery.has_won(bet)]

        for channel in client_thread_channels:
            channel.put(winner_bets)

    def start_raffle(self, lottery: Lottery, quorum_count: int, client_thread_channels: list[queue.Queue]):
        action = "start-raffle"
        if quorum_count < self.agency_quorum_min:
            return
        logger.info(
            action,
            logger.LogResult.in_progress,
            "quorum-count",
            quorum_count
        )
        self.send_winners(lottery, client_thread_channels)

    def coordinate_raffle(self, coordinator_channel: queue.Queue):
        action = "coordinate-raffle"
        logger.info(action, logger.LogResult.in_progress)

        lottery = Lottery("bets.csv")
        quorum_count = 0
        client_threads_channels = []
        while True:
            coordinate_message: CoordinateMessage = coordinator_channel.get()
            if coordinate_message.is_bets():
                self.store_bets(lottery, coordinate_message.content)
            elif coordinate_message.is_start_raffle():
                quorum_count += 1
                client_threads_channels.append(coordinate_message.content)
                self.start_raffle(lottery, quorum_count, client_threads_channels)
            elif coordinate_message.is_finished():
                coordinate_message.content.join()
            else:
                logger.error(
                    action,
                    logger.LogResult.fail,
                    "invalid message type",
                    coordinate_message.message_type
                )

    def run(self):
        action = "run"
        logger.info(action, logger.LogResult.in_progress)
        coordinator_channel = queue.Queue()
        connection_acceptor_thread = threading.Thread(target=self.accept_connections, args=(coordinator_channel,))
        connection_acceptor_thread.start()
        self.coordinate_raffle(coordinator_channel)