package synchs

import (
	"github.com/adithyabhatkajake/libe2c/log"
	msg "github.com/adithyabhatkajake/libe2c/msg/synchs"
	pb "github.com/golang/protobuf/proto"
)

// How to create and validate certificates

// NewCert creates a certificate
func NewCert(certMap map[uint64]*msg.Vote) *msg.Certificate {
	sigs := make([][]byte, len(certMap))
	ids := make([]uint64, len(certMap))
	idx := 0
	for _, v := range certMap {
		sigs[idx] = v.Signature
		ids[idx] = v.Origin
		idx++
	}
	return &msg.Certificate{Signatures: sigs, Ids: ids}
}

// IsCertValid checks if the certificate is valid for the data
func (n *SyncHS) IsCertValid(bcert *msg.BlockCertificate) bool {
	// Certificate for genesis is always correct
	if bcert.Data.Block.Data.Index == 0 {
		return true
	}
	if len(bcert.BCert.Ids) != len(bcert.BCert.Signatures) {
		log.Error("In the certificate, number of signers != number of signatures")
		return false
	}
	if uint64(len(bcert.BCert.Ids)) < n.config.GetNumberOfFaultyNodes() {
		log.Error("The certificate has < f signatures")
		return false
	}
	data, err := pb.Marshal(bcert.Data)
	if err != nil {
		log.Error("Marshalling failed during Certificate verification")
		return false
	}
	for idx, id := range bcert.BCert.Ids {
		sigOk, err := n.config.GetPubKeyFromID(id).Verify(data, bcert.BCert.Signatures[idx])
		if err != nil {
			log.Error("Certificate signature verification error")
			return false
		}
		if !sigOk {
			log.Error("Certificate signature is invalid for idx", idx)
			return false
		}
	}
	return true
}