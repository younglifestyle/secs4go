package main

import (
	"flag"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/younglifestyle/secs4go/gem"
	"github.com/younglifestyle/secs4go/hsms"
	"github.com/younglifestyle/secs4go/lib-secs2-hsms-go/pkg/ast"
)

func main() {
	addr := flag.String("addr", "127.0.0.1", "Peer address")
	port := flag.Int("port", 15000, "Peer port")
	session := flag.Int("session", 2, "HSMS session identifier")
	active := flag.Bool("active", false, "Dial the peer instead of listening")
	mdln := flag.String("mdln", "secs4go-equipment", "MDLN string reported in S1F2")
	softrev := flag.String("softrev", "1.0.0", "SOFTREV string reported in S1F2")
	flag.Parse()

	rand.Seed(time.Now().UnixNano())

	protocol := hsms.NewHsmsProtocol(*addr, *port, *active, *session, "example1-equipment")
	protocol.Timeouts().SetLinktest(45)

	handler, err := gem.NewGemHandler(gem.Options{
		Protocol:            protocol,
		DeviceType:          gem.DeviceEquipment,
		MDLN:                *mdln,
		SOFTREV:             *softrev,
		InitialControlState: gem.ControlStateEquipmentOffline,
		Logging: gem.LoggingOptions{
			Enabled:                true,
			Mode:                   hsms.LoggingModeSML,
			IncludeControlMessages: false,
		},
	})
	if err != nil {
		log.Fatalf("create GEM handler: %v", err)
	}
	defer handler.Disable()

	installCallbacks(handler)
	registerVariables(handler)
	registerRemoteCommands(handler)
	registerCollectionEvents(handler)

	handler.Enable()

	if *active {
		log.Printf("equipment dialing %s:%d session=0x%X", *addr, *port, *session)
	} else {
		log.Printf("equipment listening on %s:%d session=0x%X", *addr, *port, *session)
	}
	if !handler.WaitForCommunicating(0) {
		log.Println("waiting for COMMUNICATING state ...")
	}
	log.Println("equipment ready; press Ctrl+C to exit")

	tempTicker := time.NewTicker(5 * time.Second)
	defer tempTicker.Stop()

	for {
		select {
		case <-interrupt():
			log.Println("shutting down")
			return
		case <-tempTicker.C:
			// periodically trigger CEID 3001 if linked
			if handler.State() == gem.CommunicationStateCommunicating {
				if err := handler.TriggerCollectionEvent(int64(3001)); err != nil {
					log.Printf("trigger CEID 3001: %v", err)
				}
			}
		}
	}
}

func installCallbacks(handler *gem.GemHandler) {
	handler.Events().ControlStateChanged.AddCallback(func(data map[string]interface{}) {
		prev, _ := data["previous"].(gem.ControlState)
		next, _ := data["current"].(gem.ControlState)
		mode, _ := data["mode"].(gem.OnlineControlMode)
		log.Printf("control state changed: %s -> %s (mode=%s)", prev, next, mode)
	})

	handler.Events().HandlerCommunicating.AddCallback(func(data map[string]interface{}) {
		log.Println("HSMS connection established; GEM communicating")
	})
}

func registerVariables(handler *gem.GemHandler) {
	var temperature uint32 = 23

	tempVar, err := gem.NewStatusVariable(1001, "Temperature", "C", gem.WithStatusValueProvider(func() (ast.ItemNode, error) {
		val := atomic.AddUint32(&temperature, uint32(rand.Intn(3)-1))
		if val < 10 {
			atomic.StoreUint32(&temperature, 10)
			val = 10
		}
		return ast.NewUintNode(4, int(val)), nil
	}))
	if err != nil {
		log.Fatalf("create status variable: %v", err)
	}
	if err := handler.RegisterStatusVariable(tempVar); err != nil {
		log.Fatalf("register status variable: %v", err)
	}

	pressureVar, err := gem.NewDataVariable(2001, "Pressure", gem.WithDataValueProvider(func() (ast.ItemNode, error) {
		return ast.NewUintNode(4, 900+rand.Intn(40)), nil
	}))
	if err != nil {
		log.Fatalf("create data variable: %v", err)
	}
	if err := handler.RegisterDataVariable(pressureVar); err != nil {
		log.Fatalf("register data variable: %v", err)
	}

}

func registerCollectionEvents(handler *gem.GemHandler) {
	ce, err := gem.NewCollectionEvent(3001, "StatusSnapshot")
	if err != nil {
		log.Fatalf("create collection event: %v", err)
	}
	if err := handler.RegisterCollectionEvent(ce); err != nil {
		log.Fatalf("register collection event: %v", err)
	}
}

func registerRemoteCommands(handler *gem.GemHandler) {
	handler.SetRemoteCommandHandler(func(req gem.RemoteCommandRequest) (int, error) {
		log.Printf("remote command received: %s params=%v", req.Command, req.Parameters)
		switch req.Command {
		case "START":
			go func() {
				// Give the host a moment to set up report links
				time.Sleep(2 * time.Second)
				if err := handler.TriggerCollectionEvent(int64(3001)); err != nil {
					log.Printf("trigger CEID 3001 after START: %v", err)
				}
			}()
			return 0, nil
		case "STOP":
			return 0, nil
		default:
			return 1, nil
		}
	})
}

func interrupt() <-chan os.Signal {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	return ch
}
