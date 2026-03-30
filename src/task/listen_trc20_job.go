package task

import (
	"sync"

	"github.com/assimon/luuu/model/data"
	"github.com/assimon/luuu/model/service"
	"github.com/assimon/luuu/util/log"
)

type ListenTrc20Job struct {
}

var gListenTrc20JobLock sync.Mutex

func (r ListenTrc20Job) Run() {
	gListenTrc20JobLock.Lock()
	defer gListenTrc20JobLock.Unlock()
	log.Sugar.Debug("[ListenTrc20Job] Job triggered")
	walletAddress, err := data.GetAvailableWalletAddress()
	if err != nil {
		log.Sugar.Errorf("[ListenTrc20Job] Failed to get wallet addresses: %v", err)
		return
	}
	if len(walletAddress) <= 0 {
		log.Sugar.Debug("[ListenTrc20Job] No available wallet addresses")
		return
	}
	log.Sugar.Infof("[ListenTrc20Job] Found %d wallet addresses to monitor", len(walletAddress))
	var wg sync.WaitGroup
	for _, address := range walletAddress {
		log.Sugar.Infof("[ListenTrc20Job] Listening to address: %s", address.Address)

		wg.Add(1)
		go service.Trc20CallBack(address.Address, &wg)
	}
	wg.Wait()
	log.Sugar.Debug("[ListenTrc20Job] Job completed")
}
