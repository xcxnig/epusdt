package task

import (
	"github.com/assimon/luuu/util/log"
	"github.com/robfig/cron/v3"
)

func Start() {
	log.Sugar.Info("[task] Starting task scheduler...")
	c := cron.New()
	// trc20钱包监听
	_, err := c.AddJob("@every 5s", ListenTrc20Job{})
	if err != nil {
		log.Sugar.Errorf("[task] Failed to add ListenTrc20Job: %v", err)
		return
	}
	log.Sugar.Info("[task] ListenTrc20Job scheduled successfully (@every 5s)")
	c.Start()
	log.Sugar.Info("[task] Task scheduler started")
}
