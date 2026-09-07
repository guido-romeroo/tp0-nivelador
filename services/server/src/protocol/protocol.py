class Message:
    BET = 0x01
    WINNER = 0x02
    BYE = 0x03

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

    def is_bet(self):
        return self.type == Message.BET

    def is_bye(self):
        return self.type == Message.BYE