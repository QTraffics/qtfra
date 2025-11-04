package iolib

type NeedHandshake interface {
	NeedHandshake() bool
}

type HandshakeBuffer interface {
	Handshake(bs []byte) (n int, err error)
}

type Handshaker interface {
	Handshake() error
}
