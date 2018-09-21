package btcLikeTxDriver

import (
	"errors"
	"math/big"

	owcrypt "github.com/blocktree/go-OWCrypt"
)

type SignaturePubkey struct {
	Signature []byte
	Pubkey    []byte
}

func serilizeS(sig []byte) []byte {
	s := sig[32:]
	numS := new(big.Int).SetBytes(s)
	numHalfOrder := new(big.Int).SetBytes(HalfCurveOrder)
	if numS.Cmp(numHalfOrder) > 0 {
		numOrder := new(big.Int).SetBytes(CurveOrder)
		numS.Sub(numOrder, numS)

		return append(sig[:32], numS.Bytes()...)
	}
	return sig
}

func calcSignaturePubkey(txHash [][]byte, unlockData []TxUnlock) ([]SignaturePubkey, error) {
	if len(txHash) != len(unlockData) {
		return nil, errors.New("The number of private keys and hashes is not match!")
	}
	ret := []SignaturePubkey{}
	for i := 0; i < len(txHash); i++ {
		if unlockData[i].PrivateKey == nil || len(unlockData[i].PrivateKey) != 32 {
			return nil, errors.New("Invalid Private key!")
		}
		if txHash[i] == nil || len(txHash[i]) != 32 {
			return nil, errors.New("Invalid transaction hash data!")
		}
		sig, err := owcrypt.Signature(unlockData[i].PrivateKey, nil, 0, txHash[i], 32, owcrypt.ECC_CURVE_SECP256K1)
		if err != owcrypt.SUCCESS {
			return nil, errors.New("Signature failed!")
		}
		sig = serilizeS(sig)
		pub, err := owcrypt.GenPubkey(unlockData[i].PrivateKey, owcrypt.ECC_CURVE_SECP256K1)
		if err != owcrypt.SUCCESS {
			return nil, errors.New("Get Pubkey failed!")
		}
		pub = owcrypt.PointCompress(pub, owcrypt.ECC_CURVE_SECP256K1)
		ret = append(ret, SignaturePubkey{sig, pub})
	}
	return ret, nil
}

func (sp SignaturePubkey) encodeToScript(sigType byte) []byte {
	r := sp.Signature[:32]
	s := sp.Signature[32:]

	if r[0]&0x80 == 0x80 {
		r = append([]byte{0x00}, r...)
	}
	if s[0]&0x80 == 0x80 {
		s = append([]byte{0}, s...)
	}

	r = append([]byte{byte(len(r))}, r...)
	r = append([]byte{0x02}, r...)
	s = append([]byte{byte(len(s))}, s...)
	s = append([]byte{0x02}, s...)

	rs := append(r, s...)
	rs = append([]byte{byte(len(rs))}, rs...)
	rs = append(rs, sigType)
	rs = append([]byte{0x30}, rs...)
	rs = append([]byte{byte(len(rs))}, rs...)

	pub := append([]byte{byte(len(sp.Pubkey))}, sp.Pubkey...)

	return append(rs, pub...)
}

func decodeFromScriptBytes(script []byte) (*SignaturePubkey, error) {
	limit := len(script)
	if limit == 0 {
		return nil, errors.New("Invalid script data!")
	}

	var ret SignaturePubkey
	index := 0

	if index+1 > limit {
		return nil, errors.New("Invalid script data!")
	}
	sigLen := script[index]
	index++

	if index+1 > limit {
		return nil, errors.New("Invalid script data!")
	}
	if script[index] != 0x30 {
		return nil, errors.New("Invalid signature data!")
	}
	index++

	if index+1 > limit {
		return nil, errors.New("Invalid script data!")
	}
	rsLen := script[index]
	index++

	if index+1 > limit {
		return nil, errors.New("Invalid script data!")
	}
	if script[index] != 0x02 {
		return nil, errors.New("Invalid signature data!")
	}
	index++

	if index+1 > limit {
		return nil, errors.New("Invalid script data!")
	}
	rLen := script[index]
	index++

	if rLen == 0x21 {
		if index+2 > limit {
			return nil, errors.New("Invalid script data!")
		}
		if script[index] != 0x00 && (script[index+1]&0x80 != 0x80) {
			return nil, errors.New("Invalid signature data!")
		}
		index++
	}

	if index+32 > limit {
		return nil, errors.New("Invalid script data!")
	}
	ret.Signature = script[index : index+32]
	index += 32

	if index+1 > limit {
		return nil, errors.New("Invalid script data!")
	}
	if script[index] != 0x02 {
		return nil, errors.New("Invalid signature data!")
	}
	index++

	if index+1 > limit {
		return nil, errors.New("Invalid script data!")
	}
	sLen := script[index]
	index++

	if sLen == 0x21 {
		if index+2 > limit {
			return nil, errors.New("Invalid script data!")
		}
		if script[index] != 0x00 && (script[index+1]&0x80 != 0x80) {
			return nil, errors.New("Invalid signature data!")
		}
		index++
	}

	if index+32 > limit {
		return nil, errors.New("Invalid script data!")
	}
	ret.Signature = append(ret.Signature, script[index:index+32]...)
	index += 32

	if index+1 > limit {
		return nil, errors.New("Invalid script data!")
	}
	if script[index] != SigHashAll {
		return nil, errors.New("Only sigAll supported!")
	}
	index++

	if index+1 > limit {
		return nil, errors.New("Invalid script data!")
	}
	pubLen := script[index]
	index++
	if pubLen != 0x21 {
		return nil, errors.New("Only compressed pubkey is supported!")
	}

	if index+33 > limit {
		return nil, errors.New("Invalid script data!")
	}
	ret.Pubkey = script[index : index+33]
	index += 33

	if (rLen+sLen+4 != rsLen) || (rsLen+3 != sigLen) || (sigLen+pubLen+2 != byte(len(script))) {
		return nil, errors.New("Invalid transaction data!")
	}

	if index != len(script) {
		return nil, errors.New("Invalid transaction data!")
	}
	return &ret, nil
}
