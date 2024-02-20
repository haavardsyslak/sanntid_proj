package main

import (
	"Network-go/network/localip"
	"Network-go/network/peers"
    "sanntid/conn"
	"flag"
	"fmt"
	"os"
)

func main() {
    var id string      
    flag.StringVar(&id, "id", "", "Peer ID")
    flag.Parse()


    if id == "" {
        localIP, err := localip.LocalIP()
        if err != nil{
            fmt.Println(err)
            localIP = "DISCONNECTED"
        }
        id = fmt.Sprint("peers-%s-%d", localIP, os.Getpid())
    }

    peerUpdataCh = make(chan peers.PeerUpdate)
    peerTxEnable = make(chan bool)
}


