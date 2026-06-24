package maxclient

import "github.com/szonov/max/protocol"

const (
	OpcodePing            protocol.Opcode = 1
	OpcodeHello           protocol.Opcode = 6
	OpcodeLoginByToken    protocol.Opcode = 19
	OpcodeLogout          protocol.Opcode = 20
	OpcodeQrPasswordLogin protocol.Opcode = 115
	OpcodeQrStart         protocol.Opcode = 288
	OpcodeQrPoll          protocol.Opcode = 289
	OpcodeQrLogin         protocol.Opcode = 291
)
