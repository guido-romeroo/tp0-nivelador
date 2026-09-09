import signal
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
    SHUTDOWN = 4

    def __init__(self, message_type: int, content: tuple[list[Bet], queue.Queue] | queue.Queue | threading.Thread):
        self.message_type = message_type
        self.content = content

    def is_bets(self):
        return self.message_type == CoordinateMessage.BETS
    def is_start_raffle(self):
        return self.message_type == CoordinateMessage.START_RAFFLE

    def is_finished(self):
        return self.message_type == CoordinateMessage.FINISHED

    def is_shutdown(self):
        return self.message_type == CoordinateMessage.SHUTDOWN

class Server:
    SHUTDOWN_JOIN_TIMEOUT = 5
    def __init__(self, server_host: str, server_port: int, agency_quorum_min: int) -> None:
        self.server_host = server_host
        self.server_port = server_port
        self.agency_quorum_min = agency_quorum_min

        self.shutdown_event = threading.Event()
        self.coordinator_channel = None
        self.server_socket = None
        self.client_connections = []


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

                if not self.send_bets_to_coordinator(coordinator_channel, bets): # bloqueo
                    client_connection.send(Message(Message.NACK, b'')) 
                    continue
                client_connection.send(Message(Message.ACK, b'')) 

            winner_bets = self.receive_winners_from_coordinator(coordinator_channel, agency_id) # bloqueo
            self.send_winner_bets_to_client(client_connection, winner_bets) 
        except Exception as e:
            if not self.shutdown_event.is_set():
                logger.error(
                    action, logger.LogResult.fail, "error", str(e)
                )
        finally:
            coordinator_channel.put(CoordinateMessage(CoordinateMessage.FINISHED, threading.current_thread()))
            client_connection.close()

    def accept_connection(self, server_socket: socket.socket):
        action = "accept-connection"
        try:
            client_socket, _ = server_socket.accept()
        except Exception as e:
            logger.error(action, logger.LogResult.fail)
            raise e
        return ConnectionFacilitator(client_socket)

    def accept_connections(self, coordinator_channel: queue.Queue):
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as server_socket:
            self.server_socket = server_socket
            server_socket.bind((self.server_host, self.server_port))
            server_socket.listen()
            threads = []
            while not self.shutdown_event.is_set():
                try:
                    connection = self.accept_connection(server_socket)
                except Exception as e:
                    if self.shutdown_event.is_set():
                        break
                    logger.error(
                        "accept-connections",
                        logger.LogResult.fail,
                        "error while accepting connection",
                        str(e)
                    )
                    continue      
                self.client_connections.append(connection.connection)
                thread = threading.Thread(target=self._handle_client, args=(connection, coordinator_channel))
                thread.start()
                threads.append(thread)
            for thread in threads:
                thread.join(timeout=self.SHUTDOWN_JOIN_TIMEOUT) # aseguramos frente a un shutdown que todos los hilos terminen antes de cerrar el servidor, si no lo hizo el coordinator antes


    def store_bets(self, lottery: Lottery, content: tuple[list[Bet], queue.Queue]):
        bets = content[0]
        ack_channel = content[1]
        if self.shutdown_event.is_set():
            ack_channel.put(False)
            return
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

    def send_winners(self, lottery: Lottery, channels_to_send_winners: list[queue.Queue]):
        if self.shutdown_event.is_set():
            for channel in channels_to_send_winners:
                channel.put(None)
            return

        winner_bets = [bet for bet in lottery.load_bets() if lottery.has_won(bet)]

        for channel in channels_to_send_winners:
            channel.put(winner_bets)

    def start_raffle(self, lottery: Lottery, quorum_count: int, channels_to_send_winners: list[queue.Queue]):
        action = "start-raffle"
        if quorum_count < self.agency_quorum_min:
            return
        logger.info(
            action,
            logger.LogResult.in_progress,
            "quorum-count",
            quorum_count
        )
        self.send_winners(lottery, channels_to_send_winners)

    def graceful_shutdown(self, channels_to_send_winners: list[queue.Queue]):
        if channels_to_send_winners:
            self.send_winners(None, channels_to_send_winners) # desbloqueamos a threads de clientes
        try:
            while True:
                msg = self.coordinator_channel.get_nowait()
                if msg.is_finished():
                    msg.content.join(timeout=self.SHUTDOWN_JOIN_TIMEOUT) # aseguramos frente a un shutdown que todos los hilos terminen antes de cerrar el servidor, si no lo hizo el coordinator antes
                elif msg.is_bets():
                    msg.content[1].put(False) # desbloqueamos a threads de clientes
                elif msg.is_start_raffle():
                    msg.content.put(None) # desbloqueamos a threads de clientes
        except queue.Empty:
            logger.info(
                "graceful-shutdown",
                logger.LogResult.success,
                "all client threads are unblocked"
            )

    def coordinate_raffle(self):
        action = "coordinate-raffle"
        logger.info(action, logger.LogResult.in_progress)

        lottery = Lottery("bets.csv")
        quorum_count = 0
        channels_to_send_winners = []
        while True:
            if self.shutdown_event.is_set():
                self.graceful_shutdown(channels_to_send_winners)
                break
            coordinate_message: CoordinateMessage = self.coordinator_channel.get()
            if coordinate_message.is_shutdown():
                self.graceful_shutdown(channels_to_send_winners)
                break
            if coordinate_message.is_bets():
                self.store_bets(lottery, coordinate_message.content)
            elif coordinate_message.is_start_raffle():
                quorum_count += 1
                if quorum_count <= self.agency_quorum_min:
                    channels_to_send_winners.append(coordinate_message.content)
                else:
                    channels_to_send_winners = [coordinate_message.content]
                self.start_raffle(lottery, quorum_count, channels_to_send_winners)
            elif coordinate_message.is_finished():
                coordinate_message.content.join()

            else:
                logger.error(
                    action,
                    logger.LogResult.fail,
                    "invalid message type",
                    coordinate_message.message_type
                )

    def handle_shutdown_signal(self, _signum, _frame):
        self.shutdown_event.set()
        self.coordinator_channel.put(CoordinateMessage(CoordinateMessage.SHUTDOWN, None)) # desbloqueamos al coordinator si lo estaba
        if self.server_socket is not None:
            try:
                self.server_socket.shutdown(socket.SHUT_RDWR)
            except OSError:
                pass
            self.server_socket.close()
        for connection in self.client_connections:
            if connection is not None:
                try:
                    connection.shutdown(socket.SHUT_RDWR)
                except OSError:
                    pass
                connection.close()

    def run(self):
        signal.signal(signal.SIGTERM, self.handle_shutdown_signal)
        action = "run"
        self.coordinator_channel = queue.Queue()
        connection_acceptor_thread = threading.Thread(target=self.accept_connections, args=(self.coordinator_channel,))
        connection_acceptor_thread.start()
        logger.info(action, logger.LogResult.in_progress)
        self.coordinate_raffle()
        connection_acceptor_thread.join(timeout=self.SHUTDOWN_JOIN_TIMEOUT)