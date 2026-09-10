from lottery.bet import Bet 

class Message:
    BETS = 0x01
    WINNER = 0x02
    ACK = 0x03
    NACK = 0x04
    BYE = 0x05

    def __init__(self, message_type, payload):
        self.type = message_type
        self.payload = payload

    def to_bytes(self):
        return bytes([self.type]) + self.payload

    @classmethod
    def from_bytes(cls, data):
        if not data:
            raise ValueError("empty message")

        return cls(
            data[0],
            data[1:]
        )

    def is_bets(self):
        return self.type == Message.BETS

    def is_bye(self):
        return self.type == Message.BYE

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
        bet.number.to_bytes(4, byteorder="big")
    )

def bets_from_bytes(data) -> list[Bet]:
    min_size = 2 + 2 + 2 + 4 + 2 + 2
    if len(data) < min_size:
        raise ValueError("data is too short")

    bets = []
    offset = 0
    while offset < len(data):
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

        if len(data) < offset + 4:
            raise ValueError("data is too short to contain number")

        number = int.from_bytes(data[offset:offset + 4], byteorder="big")
        offset += 4
        bets.append(Bet(
            agency_id=agency_id,
            first_name=first_name,
            last_name=last_name,
            document=document,
            birthdate=birthdate,
            number=number,
        ))

    return bets

