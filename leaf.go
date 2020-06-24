package leaf

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/luyu6056/leaf/cluster"
	"github.com/luyu6056/leaf/conf"
	"github.com/luyu6056/leaf/console"
	"github.com/luyu6056/leaf/log"
	"github.com/luyu6056/leaf/module"
)

var serverName = ""

func RunWithName(name string, mods ...module.Module) {
	serverName = name
	Run(mods...)
}

func Run(mods ...module.Module) {
	// logger
	if conf.LogLevel != "" {
		logger, err := log.New(conf.LogLevel, conf.LogPath, conf.LogFlag)
		if err != nil {
			panic(err)
		}
		log.Export(logger)
		defer logger.Close()
	}

	// module
	for i := 0; i < len(mods); i++ {
		module.Register(mods[i])
	}
	module.Init()

	// cluster
	cluster.Init()

	// console
	console.Init()

	log.Release(">>>>>>>>>>>>>>>> %s Leaf starting up! <<<<<<<<<<<<<<<<<", serverName)

	// close
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGQUIT)
	sig := <-c
	log.Release("Leaf closing down (signal: %v)", sig)
	Stop()
}
func Stop() {
	console.Destroy()
	cluster.Destroy()
	module.Destroy()
	log.Release("Leaf closed")
	os.Exit(1)
}
