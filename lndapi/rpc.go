package lndapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/lnrpc/routerrpc"
	"github.com/lightningnetwork/lnd/lntypes"
	"github.com/lightningnetwork/lnd/lnwire"
	"github.com/lightningnetwork/lnd/record"
)

func AddInvoice(lcli lnrpc.LightningClient, assetId uint32, amount int64, expire_day int64) (*lnrpc.AddInvoiceResponse, error) {

	valueMsat := int64(0)
	amt := int64(0)
	if assetId > 0 {
		amt = amount
	} else {
		valueMsat = amount * 1000
	}
	in := &lnrpc.Invoice{AssetId: assetId, ValueMsat: valueMsat, Amount: amt, Expiry: 3600 * 24 * expire_day, Refundable: true}
	return lcli.OB_AddInvoice(context.TODO(), in)
}

func SendpaymentRefund(lcli lnrpc.LightningClient, rcli routerrpc.RouterClient, rawInvoice, refundAddress string) (bool, error) {
	decodeReq := &lnrpc.PayReqString{PayReq: rawInvoice}
	decodeResp, err := lcli.DecodePayReq(context.TODO(), decodeReq)
	if err != nil {
		return false, err
	}
	req := &routerrpc.SendPaymentRequest{
		AssetId:           decodeResp.AssetId,
		DestCustomRecords: make(map[uint64][]byte),
	}

	invoiceAmt := decodeResp.GetAmtMsat()
	req.AmtMsat = decodeResp.GetAmtMsat()
	if req.AssetId != lnwire.BtcAssetId {
		invoiceAmt = decodeResp.GetAmount()
		req.AssetAmt = decodeResp.GetAmount()
	}

	dest, _ := hex.DecodeString(refundAddress)
	req.Dest = dest

	var preimage lntypes.Preimage
	if _, err := rand.Read(preimage[:]); err != nil {
		return false, err
	}
	req.DestCustomRecords[record.KeySendType] = preimage[:]
	hash := preimage.Hash()
	rHash := hash[:]
	req.PaymentHash = rHash

	// Calculate fee limit based on the determined amount.
	feeLimit := int64(defaultRoutingFeeLimitForAmount(invoiceAmt))
	req.FeeLimitMsat = feeLimit

	req.NoInflightUpdates = true
	req.TimeoutSeconds = 5
	stream, err := rcli.OB_SendPaymentV2(context.TODO(), req)
	if err != nil {
		return false, err
	}
	payment, err := stream.Recv()
	if err != nil {
		return false, err
	}
	if payment.Status == lnrpc.Payment_SUCCEEDED {
		return true, nil
	} else {
		return false, errors.New(fmt.Sprintf("err pay invoice with status %v", payment.Status))
	}
	return false, nil
}
func Sendpayment(lcli lnrpc.LightningClient, rcli routerrpc.RouterClient, payRequst string) (bool, error) {
	decodeReq := &lnrpc.PayReqString{PayReq: payRequst}
	decodeResp, err := lcli.DecodePayReq(context.TODO(), decodeReq)
	if err != nil {
		return false, err
	}
	req := &routerrpc.SendPaymentRequest{
		AssetId:           decodeResp.AssetId,
		PaymentRequest:    payRequst,
		DestCustomRecords: make(map[uint64][]byte),
	}
	invoiceAmt := decodeResp.GetAmtMsat()
	//req.AmtMsat = decodeResp.GetAmtMsat()
	if req.AssetId != lnwire.BtcAssetId {
		invoiceAmt = decodeResp.GetAmount()
		//req.AssetAmt = decodeResp.GetAmount()
	}

	// Calculate fee limit based on the determined amount.
	feeLimit := int64(defaultRoutingFeeLimitForAmount(invoiceAmt))
	req.FeeLimitMsat = feeLimit

	req.NoInflightUpdates = true
	req.TimeoutSeconds = 5
	stream, err := rcli.OB_SendPaymentV2(context.TODO(), req)
	if err != nil {
		return false, err
	}
	payment, err := stream.Recv()
	if err != nil {
		return false, err
	}
	if payment.Status == lnrpc.Payment_SUCCEEDED {
		return true, nil
	} else {
		return false, errors.New(fmt.Sprintf("err pay invoice with status %v", payment.Status))
	}
}

func defaultRoutingFeeLimitForAmount(a int64) int64 {
	// Allow 100% fees up to a certain amount to accommodate for base fees.
	if a <= RoutingFee100PercentUpTo {
		return a
	}

	// Everything larger than the cut-off amount will get a default fee
	// percentage.
	return a * DefaultRoutingFeePercentage / 100
}

const (
	RoutingFee100PercentUpTo          = 1000000
	DefaultRoutingFeePercentage int64 = 5
)
