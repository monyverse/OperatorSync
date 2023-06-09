package main

import (
	"github.com/Crossbell-Box/OperatorSync/app/worker/config"
	"github.com/Crossbell-Box/OperatorSync/app/worker/global"
	"github.com/Crossbell-Box/OperatorSync/app/worker/inits"
	commonInits "github.com/Crossbell-Box/OperatorSync/common/inits"
	"log"
)

func main() {
	// Initialize everything
	log.Println("Initializing...")

	// Initialize config
	if err := inits.Config(); err != nil {
		log.Fatalln("Failed to load config: ", err)
	}

	// Initialize logger
	if err := inits.Logger(); err != nil {
		log.Fatalln("Failed to load logger: ", err)
	}

	global.Logger.Info("Logger initialized, switch to here.")

	// Initialize redis
	if err := commonInits.Redis(config.Config.RedisConnString); err != nil {
		global.Logger.Fatal("Failed to load redis: ", err.Error())
	}

	// Initialize MQ
	if err := commonInits.MQ(config.Config.MQConnString, global.Logger); err != nil {
		global.Logger.Fatal("Failed to load MQ: ", err.Error())
	}

	// Initialize RPC
	if err := inits.RPC(); err != nil {
		global.Logger.Fatal("Failed to start RPC server: ", err.Error())
	}

	global.Logger.Debug("RPC server initialized, ready to accept connections.")

	// Initialize jobs
	if err := inits.Jobs(); err != nil {
		global.Logger.Fatal("Failed to start jobs: ", err.Error())
	}

	global.Logger.Info("Worker started!")

	select {} // Keep process running

}
