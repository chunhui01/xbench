package cases

import (
	"errors"
	"sync"
	"github.com/xuperchain/xuperbench/adapter/xchain/lib"
	"github.com/xuperchain/xuperbench/common"
	"github.com/xuperchain/xuperbench/log"
	"github.com/xuperchain/xuperunion/pb"
)

type Deal struct {
	common.TestCase
}

type ch chan *pb.TxStatus

var (
	txstore = []ch{}
	wg = sync.WaitGroup{}
)

func createtx(i int, batch int, chain string) {
	for c:=0; c<batch; c++ {
		tx := lib.ProfTx(Accts[i], Bank.Address, Clis[i])
		if i == 0 && c > 0 && c % 500 == 0 {
			log.DEBUG.Printf("prepare Tx in progress %d", c)
		}
		txstore[i] <- tx
	}
	wg.Done()
}

// In this case, we run perfomance test with Transactions which
// are generated and signed beforehand.

// Init implements the comm.IcaseFace
func (d Deal) Init(args ...interface{}) error {
	parallel := args[0].(int)
	env := args[1].(common.TestEnv)
	lib.SetCrypto(env.Crypto)
	amount := env.Batch
	Bank = lib.InitBankAcct("")
	addrs := []string{}
	for i:=0; i<parallel; i++ {
		Accts[i], _ = lib.CreateAcct(env.Crypto)
		addrs = append(addrs, Accts[i].Address)
		if len(Clis) < parallel {
			cli := lib.Conn(env.Host, env.Chain)
			Clis = append(Clis, cli)
		}
	}
	lib.InitIdentity(Bank, addrs, Clis[0])
	txstore = make([]ch, parallel)
	wg.Add(parallel)
	for i, _ := range txstore {
		txstore[i] = make(chan *pb.TxStatus, amount)
	}
	log.INFO.Printf("prepare tokens of test accounts ...")
	for i := range Accts {
		rsp, _, err := lib.Transplit(Bank, Accts[i].Address, amount, Clis[0])
		if rsp.Header.Error != 0 || err != nil {
			log.ERROR.Printf("init token error: %#v", err)
			return errors.New("init token error")
		}
	}
	log.INFO.Printf("prepere tx of test accounts ...")
	for k := range Accts {
		go createtx(k, amount, env.Chain)
	}
	wg.Wait()
	log.INFO.Printf("init done ...")
	return nil
}

// Run implements the comm.IcaseFace
func (d Deal) Run(seq int, args ...interface{}) error {
	txs := <-txstore[seq]
	rsp, _, err := Clis[seq].PostTx(txs)
	if rsp.Header.Error != 0 || err != nil {
		return errors.New("run posttx error")
	}
	return nil
}

// End implements the comm.IcaseFace
func (d Deal) End(args ...interface{}) error {
	log.INFO.Printf("Deal perf-test done.")
	return nil
}
