package wrtc

import (
	"crypto/ecdh"
	"crypto/ed25519"
	"encoding/json"

	"github.com/pion/webrtc/v4"
)

type Peer struct {
	pc *webrtc.PeerConnection
}

func BuildConnReq(pubkey *ecdh.PublicKey, pubsign ed25519.PublicKey) []byte {
	out := make([]byte, len(pubsign)+len(pubkey.Bytes()))
	pos := 0
	pos += copy(out[pos:], pubsign)
	copy(out[pos:], pubkey.Bytes())
	return out
}

func Setup() (*Peer, error) {
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	}
	pc, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return nil, err
	}

	return &Peer{pc: pc}, nil
}

func BuildOffer(pc *Peer) ([]byte, error) {
	offerSDP, err := pc.pc.CreateOffer(nil)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(offerSDP)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func BuildAnswer(input []byte, pc Peer) ([]byte, error) {
	var offerSDP webrtc.SessionDescription
	json.Unmarshal(input, &offerSDP)
	err := pc.pc.SetRemoteDescription(offerSDP)
	if err != nil {
		return nil, err
	}
	answerSDP, err := pc.pc.CreateAnswer(nil)
	err = pc.pc.SetLocalDescription(answerSDP)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(answerSDP)
	if err != nil {
		return nil, err
	}
	return data, err
}
