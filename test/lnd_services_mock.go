package test

import (
	"errors"
	"sync"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/wire"
	"github.com/lightninglabs/loop/lndclient"
	"github.com/lightningnetwork/lnd/chainntnfs"
	"github.com/lightningnetwork/lnd/lntypes"
	"github.com/lightningnetwork/lnd/lnwallet/chainfee"
	"github.com/lightningnetwork/lnd/zpay32"
)

var (
	testStartingHeight = int32(600)
	testNodePubkey     = "03f5374b16f0b1f1b49101de1b9d89e0b460bc57ce9c2f9" +
		"132b73dfc76d3704daa"
	testSignature = []byte{55, 66, 77, 88, 99}
)

// NewMockLnd returns a new instance of LndMockServices that can be used in unit
// tests.
func NewMockLnd() *LndMockServices {
	lightningClient := &mockLightningClient{}
	walletKit := &mockWalletKit{
		feeEstimates: make(map[int32]chainfee.SatPerKWeight),
	}
	chainNotifier := &mockChainNotifier{}
	signer := &mockSigner{}
	invoices := &mockInvoices{}
	router := &mockRouter{}

	lnd := LndMockServices{
		LndServices: lndclient.LndServices{
			WalletKit:     walletKit,
			Client:        lightningClient,
			ChainNotifier: chainNotifier,
			Signer:        signer,
			Invoices:      invoices,
			Router:        router,
			ChainParams:   &chaincfg.TestNet3Params,
		},
		SendPaymentChannel:           make(chan PaymentChannelMessage),
		ConfChannel:                  make(chan *chainntnfs.TxConfirmation),
		RegisterConfChannel:          make(chan *ConfRegistration),
		RegisterSpendChannel:         make(chan *SpendRegistration),
		SpendChannel:                 make(chan *chainntnfs.SpendDetail),
		TxPublishChannel:             make(chan *wire.MsgTx),
		SendOutputsChannel:           make(chan wire.MsgTx),
		SettleInvoiceChannel:         make(chan lntypes.Preimage),
		SingleInvoiceSubcribeChannel: make(chan *SingleInvoiceSubscription),

		RouterSendPaymentChannel: make(chan RouterPaymentChannelMessage),
		TrackPaymentChannel:      make(chan TrackPaymentMessage),

		FailInvoiceChannel: make(chan lntypes.Hash, 2),
		epochChannel:       make(chan int32),
		Height:             testStartingHeight,
		NodePubkey:         testNodePubkey,
		Signature:          testSignature,
	}

	lightningClient.lnd = &lnd
	chainNotifier.lnd = &lnd
	walletKit.lnd = &lnd
	invoices.lnd = &lnd
	router.lnd = &lnd
	signer.lnd = &lnd

	lnd.WaitForFinished = func() {
		chainNotifier.WaitForFinished()
		lightningClient.WaitForFinished()
		invoices.WaitForFinished()
	}

	return &lnd
}

// PaymentChannelMessage is the data that passed through SendPaymentChannel.
type PaymentChannelMessage struct {
	PaymentRequest string
	Done           chan lndclient.PaymentResult
}

// TrackPaymentMessage is the data that passed through TrackPaymentChannel.
type TrackPaymentMessage struct {
	Hash lntypes.Hash

	Updates chan lndclient.PaymentStatus
	Errors  chan error
}

// RouterPaymentChannelMessage is the data that passed through RouterSendPaymentChannel.
type RouterPaymentChannelMessage struct {
	lndclient.SendPaymentRequest

	TrackPaymentMessage
}

// SingleInvoiceSubscription contains the single invoice subscribers
type SingleInvoiceSubscription struct {
	Hash   lntypes.Hash
	Update chan lndclient.InvoiceUpdate
	Err    chan error
}

// LndMockServices provides a full set of mocked lnd services.
type LndMockServices struct {
	lndclient.LndServices

	SendPaymentChannel   chan PaymentChannelMessage
	SpendChannel         chan *chainntnfs.SpendDetail
	TxPublishChannel     chan *wire.MsgTx
	SendOutputsChannel   chan wire.MsgTx
	SettleInvoiceChannel chan lntypes.Preimage
	FailInvoiceChannel   chan lntypes.Hash
	epochChannel         chan int32

	ConfChannel          chan *chainntnfs.TxConfirmation
	RegisterConfChannel  chan *ConfRegistration
	RegisterSpendChannel chan *SpendRegistration

	SingleInvoiceSubcribeChannel chan *SingleInvoiceSubscription

	RouterSendPaymentChannel chan RouterPaymentChannelMessage
	TrackPaymentChannel      chan TrackPaymentMessage

	Height     int32
	NodePubkey string
	Signature  []byte

	WaitForFinished func()

	lock sync.Mutex
}

// NotifyHeight notifies a new block height.
func (s *LndMockServices) NotifyHeight(height int32) error {
	s.Height = height

	select {
	case s.epochChannel <- height:
	case <-time.After(Timeout):
		return ErrTimeout
	}
	return nil
}

// IsDone checks whether all channels have been fully emptied. If not this may
// indicate unexpected behaviour of the code under test.
func (s *LndMockServices) IsDone() error {
	select {
	case <-s.SendPaymentChannel:
		return errors.New("SendPaymentChannel not empty")
	default:
	}

	select {
	case <-s.SpendChannel:
		return errors.New("SpendChannel not empty")
	default:
	}

	select {
	case <-s.TxPublishChannel:
		return errors.New("TxPublishChannel not empty")
	default:
	}

	select {
	case <-s.SendOutputsChannel:
		return errors.New("SendOutputsChannel not empty")
	default:
	}

	select {
	case <-s.SettleInvoiceChannel:
		return errors.New("SettleInvoiceChannel not empty")
	default:
	}

	select {
	case <-s.ConfChannel:
		return errors.New("ConfChannel not empty")
	default:
	}

	select {
	case <-s.RegisterConfChannel:
		return errors.New("RegisterConfChannel not empty")
	default:
	}

	select {
	case <-s.RegisterSpendChannel:
		return errors.New("RegisterSpendChannel not empty")
	default:
	}

	return nil
}

// DecodeInvoice decodes a payment request string.
func (s *LndMockServices) DecodeInvoice(request string) (*zpay32.Invoice,
	error) {

	return zpay32.Decode(request, s.ChainParams)
}

func (s *LndMockServices) SetFeeEstimate(confTarget int32,
	feeEstimate chainfee.SatPerKWeight) {

	s.WalletKit.(*mockWalletKit).feeEstimates[confTarget] = feeEstimate
}
