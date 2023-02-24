package toolrpc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/btcsuite/btcd/btcec"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/lnrpc/routerrpc"
	"om-rpc-tool/lndapi"
	"om-rpc-tool/signal"

	//"github.com/btcsuite/btcd/btcec/v2/ecdsa"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"log"
	"strconv"
	"sync"
	"time"
)

func GetUserIdKey(ctx context.Context) (userid string, err error) {
	tlsPubKey, err := getCertPub(ctx)
	if err != nil {
		return "0", err
	}
	un := &UserNode{TlsPubKeyPre: tlsPubKey[:15]}
	err = db.First(un, un).Error
	if err != nil {
		return "0", status.Error(codes.Unauthenticated, err.Error())
	}
	return strconv.Itoa(int(un.ID)), nil
}

func getCtxUserid(ctx context.Context) int64 {
	useridStr, err := GetGrpcHeader(ctx, "userid")
	if err != nil {
		log.Println("getCtxUserid err :", err)
	}
	userid, _ := strconv.Atoi(useridStr)
	return int64(userid)
}

type UserNode struct {
	ID uint `gorm:"primarykey"`
	//online:1  offline:2
	Online       int8
	CreatedAt    time.Time
	UpdatedAt    time.Time
	TlsPubKey    string
	TlsPubKeyPre string `gorm:"index"`
	UserIdKey    string
	Alias        string
}

type LuckPkServer struct {
	//UnimplementedLuckPkApiServer
	//btcClient *rpcclient.Client
	////regtest testnet mainnet
	//NetType string
	lndCli      lnrpc.LightningClient
	routerCli   routerrpc.RouterClient
	shudownChan *signal.Interceptor
}

func (l *LuckPkServer) MonitorInvoice() {
	log.Println("begin MonitorInvoice")
	for {
		rs, serr := l.lndCli.SubscribeInvoices(context.TODO(), &lnrpc.InvoiceSubscription{AddIndex: 1000000})
		if serr == nil {
			log.Println("MonitorInvoice with clientid")
		LOOP:
			for {
				select {
				case <-l.shudownChan.ShutdownChannel():
					return
				default:
					invoice, err := rs.Recv()
					if err != nil {
						serr = err
						break LOOP
					}
					log.Printf("rev invoice : %x %s", invoice.RHash, invoice.State)
					if invoice.State == lnrpc.Invoice_SETTLED {
						//spay
						l.updateSpayByInvoide(invoice)
						//start luckpack
						updateLuckPkByInvoide(invoice)
					}
				}
			}
			if serr != nil {
				log.Println("MonitorInvoice err", serr)
			}
			time.Sleep(5 * time.Second)
		}
	}
}
func (l *LuckPkServer) runUserSpay(userId int64) {
	spays := []*Spay{}
	err := db.Find(&spays, "user_id=? and status=?", userId, SpayStatus_UserPayed).Error
	if err == nil {
		for _, spay := range spays {
			_, err := lndapi.Sendpayment(l.lndCli, l.routerCli, spay.UserInvoice)
			if err != nil {
				db.Model(spay).Updates(Spay{ErrMsg: err.Error(), ErrTimes: spay.ErrTimes + 1})
				log.Println("server pay user invoice err", err)
				return
			}
			err = db.Model(spay).Updates(Spay{Status: SpayStatus_PayEnd}).Error
			if err != nil {
				return
			}
		}
	}
}
func (l *LuckPkServer) updateSpayByInvoide(invoice *lnrpc.Invoice) {
	rhash := hex.EncodeToString(invoice.RHash)
	sp := &Spay{SiPayHash: rhash}
	err := db.First(sp, sp).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return
	} else if err != nil {
		log.Println("updateSpayByInvoide db err", err)
		return
	}
	if sp.Status == SpayStatus_PayINIT {
		spState := SpayStatus_PayINIT
		msg := ""
		if invoice.State == lnrpc.Invoice_SETTLED {
			spState = SpayStatus_UserPayed
		} else if invoice.State == lnrpc.Invoice_CANCELED {
			spState = SpayStatus_Error
			msg = "Invoice_CANCELED"
		} else {
			return
		}
		err = db.Model(sp).Updates(Spay{Status: spState, ErrMsg: msg}).Error
		if err != nil {
			log.Println("updateLuckPkByInvoide db err", err)
			return
		}
		//server try pay user invoice, user may offline; when user online can trigger this pay too
		if sp.Status == SpayStatus_UserPayed {
			_, err := lndapi.Sendpayment(l.lndCli, l.routerCli, sp.UserInvoice)
			if err != nil {
				log.Println("server try pay user invoice err", err)
				return
			}
			err = db.Model(sp).Updates(Spay{Status: SpayStatus_PayEnd}).Error
			if err != nil {
				return
			}
		}
	}
}
func updateLuckPkByInvoide(invoice *lnrpc.Invoice) {
	rhash := hex.EncodeToString(invoice.RHash)
	lk := &LuckPk{PaymentHash: rhash}
	err := db.First(lk, lk).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return
	} else if err != nil {
		log.Println("updateLuckPkByInvoide db err", err)
		return
	}

	if lk.Status == LuckPKStatus_INIT {
		lkState := LuckPKStatus_INIT
		msg := ""
		if invoice.State == lnrpc.Invoice_SETTLED {
			lkState = LuckPKStatus_WorkIng
		} else if invoice.State == lnrpc.Invoice_CANCELED {
			lkState = LuckPKStatus_ErrorCreate
			msg = "Invoice_CANCELED"
		} else {
			return
		}
		err = db.Model(lk).Updates(LuckPk{Status: lkState, ErrorCreateMsg: msg}).Error
		if err != nil {
			log.Println("updateLuckPkByInvoide db err", err)
		}
	}
}

func NewLuckPkServer(nodeAddress, netType, lndDir string, shudownChan *signal.Interceptor) *LuckPkServer {
	lserver := new(LuckPkServer)
	lserver.shudownChan = shudownChan
	var err error
	lserver.lndCli, err = lndapi.GetLndClient(nodeAddress, netType, lndDir)
	if err != nil {
		panic(err)
	}
	//test State
	res, err := lserver.lndCli.OB_GetInfo(context.TODO(), &lnrpc.GetInfoRequest{})
	if err != nil {
		log.Fatal(err)
	}
	if !res.SyncedToChain {
		log.Println("warn: lndCli SyncedToChain status:", res.SyncedToChain)
	} else {
		log.Printf("lndCli %v %v SyncedToChain status ok", nodeAddress, netType)
	}
	lserver.routerCli, err = lndapi.GetRouterClient(nodeAddress, netType, lndDir)
	if err != nil {
		panic(err)
	}

	go func() {
		lserver.MonitorInvoice()
	}()
	return lserver
}

type HeartBeat struct {
	ID          uint `gorm:"primarykey"`
	CreatedAt   time.Time
	OffLineTime time.Time
	OnlineSecs  int64
	UserID      int64
}

func (l *LuckPkServer) HeartBeat(recStream LuckPkApi_HeartBeatServer) error {
	select {
	//if shutdown, new stream-connect will skip
	case <-l.shudownChan.ShutdownChannel():
		return errors.New("server is shudowning")
	default:
		userId := getCtxUserid(recStream.Context())
		hb := &HeartBeat{
			UserID:    userId,
			CreatedAt: time.Now(),
		}
		db.Save(hb)

		//update online status
		un := new(UserNode)
		db.First(un, userId)
		if un.ID > 0 {
			db.Model(un).Updates(UserNode{Online: 1})
			log.Printf("trigger user spay %v %v", un.ID, un.UserIdKey)
			l.runUserSpay(userId)
		}
		defer func() {
			hb.OffLineTime = time.Now()
			hb.OnlineSecs = int64(time.Now().Sub(hb.CreatedAt).Seconds())
			db.Save(hb)
			if un.ID > 0 {
				db.Model(un).Updates(UserNode{Online: 2})
			}
		}()
		for {
			select {
			case <-l.shudownChan.ShutdownChannel():
				return nil
			default:
				_, err := recStream.Recv()
				if err != nil {
					return err
				}
			}
		}
	}
}

func (l *LuckPkServer) RegistTlsKey(ctx context.Context, obj *RegistTlsKeyReq) (*emptypb.Empty, error) {
	//log.Println(getCertPub(ctx))
	tlsPubKey, err := getCertPub(ctx)
	if err != nil {
		return &emptypb.Empty{}, err
	}
	un := &UserNode{TlsPubKeyPre: tlsPubKey[:15]}
	err = db.First(un, un).Error
	if err == nil {
		log.Println("RegistTlsKey exists")
		if un.Alias != obj.Alias {
			un.Alias = obj.Alias
			db.Save(un)
		}
		return &emptypb.Empty{}, err
	}

	log.Printf("RegistTlsKey receive:  %x %s", obj.UserNodeKey, tlsPubKey)
	pubKey, err := btcec.ParsePubKey(obj.UserNodeKey, btcec.S256())
	if err != nil {
		return &emptypb.Empty{}, err
	}
	//sig, err := ecdsa.ParseDERSignature(obj.Sig)
	sig, err := btcec.ParseDERSignature(obj.Sig, btcec.S256())
	if err != nil {
		return &emptypb.Empty{}, err
	}
	now := time.Now().Truncate(time.Minute * 5)
	hashTxt := strconv.Itoa(int(now.Unix()))
	s256 := sha256.New()
	s256.Write([]byte(hashTxt))
	s256.Write(obj.UserNodeKey)
	hash := s256.Sum(nil)
	if !sig.Verify(hash, pubKey) {
		log.Println("RegistTlsKey Verify fail")
		return &emptypb.Empty{}, errors.New("RegistTlsKey Verify fail")
	}
	userIdKey := hex.EncodeToString(obj.UserNodeKey)
	log.Printf("RegistTlsKey ok  %s %s", userIdKey, tlsPubKey)

	un.TlsPubKey = tlsPubKey
	un.UserIdKey = userIdKey
	un.Alias = obj.Alias
	err = db.Save(un).Error
	if err != nil {
		return &emptypb.Empty{}, err
	}
	return &emptypb.Empty{}, nil
}

var db *gorm.DB

func init() {
	var err error
	db, err = gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	// Migrate the schema
	db.AutoMigrate(&UserNode{}, &LuckPk{}, &LuckItem{}, &HeartBeat{}, Spay{})
	db = db.Debug()
}

func (l *LuckPkServer) CreateSpay(ctx context.Context, sy *Spay) (*Spay, error) {
	userInvoice, err := l.lndCli.DecodePayReq(context.TODO(), &lnrpc.PayReqString{PayReq: sy.UserInvoice})
	if err != nil {
		return nil, err
	}
	//todo check server balance

	amt := userInvoice.Amount
	if userInvoice.AssetId == 0 {
		amt = userInvoice.AmtMsat / 1000
	}

	servInvoice, err := lndapi.AddInvoice(l.lndCli, uint32(userInvoice.AssetId), amt, 1)
	if err != nil {
		return nil, err
	}
	sy.ServInvoice = servInvoice.PaymentRequest
	sy.SiPayHash = hex.EncodeToString(servInvoice.RHash)
	sy.UserId = getCtxUserid(ctx)
	err = db.Save(sy).Error
	return sy, err
}
func (l *LuckPkServer) GetLuckPkInfo(ctx context.Context, obj *LuckpkIdObj) (*LuckPk, error) {
	lk := new(LuckPk)
	err := db.First(lk, "id=?", obj.Id).Error
	return lk, err
}

func (l *LuckPkServer) ListLuckItem(ctx context.Context, obj *LuckpkIdObj) (*ListLuckItemRes, error) {
	items := []*LuckItem{}
	err := db.Find(&items, "luckpk_id=?", obj.Id).Error
	if err != nil {
		return nil, err
	}
	res := new(ListLuckItemRes)
	res.Items = items
	return res, nil
}

func (l *LuckPkServer) ListLuckPk(ctx context.Context, req *ListLuckPkReq) (*ListLuckPkRes, error) {
	res := new(ListLuckPkRes)
	dbqeury := db.Model(LuckPk{}).Where("user_id=?", getCtxUserid(ctx))
	err := dbqeury.Count(&res.Count).Error
	if err == nil {
		items := []*LuckPk{}
		err = dbqeury.Find(&items).Error
		res.Items = items
	}
	return res, err
}

func (l *LuckPkServer) CreateLuckPk(ctx context.Context, pk *LuckPk) (*LuckPk, error) {
	res, err := lndapi.AddInvoice(l.lndCli, uint32(pk.AssetId), int64(pk.Amt), 1)
	if err != nil {
		return nil, err
	}
	pk.Invoice = res.PaymentRequest
	pk.PaymentHash = hex.EncodeToString(res.RHash)
	pk.UserId = getCtxUserid(ctx)
	pk.Balance = pk.Amt
	err = db.Create(pk).Error
	return pk, err
}

func (l *LuckPkServer) startupLuckPk(ctx context.Context, pk *LuckPk) (*LuckPk, error) {
	pk.Status = LuckPKStatus_WorkIng
	db.Save(pk)
	return pk, nil
}

var payLock sync.Mutex

func (l *LuckPkServer) GiveLuckPk(ctx context.Context, req *GiveLuckPkReq) (*emptypb.Empty, error) {
	payLock.Lock()
	defer payLock.Unlock()
	lk := new(LuckPk)
	err := db.First(lk, "id=?", req.Id).Error
	if err != nil {
		return nil, err
	}

	if lk.Status != LuckPKStatus_WorkIng {
		return nil, errors.New(fmt.Sprintf("luckPacket status is %s %s", lk.Status, lk.ErrorCreateMsg))
	}

	if lk.Gives >= lk.Parts {
		return nil, errors.New(fmt.Sprintf("exceed　luckPacket parts,the max part is %v", lk.Parts))
	}
	var (
		amt = int64(0)
	)
	// decode
	payreq, err := l.lndCli.DecodePayReq(context.TODO(), &lnrpc.PayReqString{PayReq: req.Invoice})
	if err != nil {
		return nil, err
	}
	amt = payreq.Amount
	if payreq.AssetId == 0 {
		amt = payreq.AmtMsat / 1000
	}
	//veryfy lk
	if int64(lk.Balance)-amt < 0 {
		return nil, errors.New("luckPackge balance insufficient")
	}
	if lk.AssetId != uint64(payreq.AssetId) {
		return nil, fmt.Errorf("missmatch assetid %v %v", payreq.AssetId, lk.AssetId)
	}
	//pay

	_, err = lndapi.Sendpayment(l.lndCli, l.routerCli, req.Invoice)
	if err != nil {
		log.Println("Sendpayment err:", err)
		return nil, err
	}

	//pk ok
	litem := new(LuckItem)
	litem.UserId = getCtxUserid(ctx)
	litem.LuckpkId = req.Id
	litem.Amt = int64(amt)
	db.Save(litem)

	lk.Gives += 1
	lk.Balance -= uint64(amt)
	if lk.Gives == lk.Parts {
		lk.Status = LuckPKStatus_End
	}
	err = db.Save(lk).Error
	return &emptypb.Empty{}, err
}

func (l *LuckPkServer) mustEmbedUnimplementedLuckPkApiServer() {
	//TODO implement me
	panic("implement me")
}
