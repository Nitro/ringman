package ringman

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Nitro/sidecar/catalog"
	"github.com/Nitro/sidecar/receiver"
	"github.com/Nitro/sidecar/service"
	"github.com/relistan/go-director"
	log "github.com/sirupsen/logrus"
)

const (
	DefaultReceiverCapacity = 50
)

// A SidecarRing is a ring backed by service discovery from Sidecar
// https://github.com/Nitro/sidecar . Sidecar itself uses Memberlist under
// the covers, but layers a lot more on top. Sidecar takes care of all
// the work of managing and bootstrapping the cluster so we don't need
// to know anything about cluster seeds. This service is expected to
// subscribe to Sidecar events, however, and uses a Sidecar Receiver to
// process them.
type SidecarRing struct {
	Manager       *HashRingManager
	managerLooper director.Looper
	sidecarUrl    string
	svcName       string
	svcPort       int64
	rcvr          *receiver.Receiver
	nodes         map[string]struct{} // Tracking which nodes we already know about
}

// NewSidecarRing returns a properly configured SidecarRing that will filter
// incoming changes by the service name provided and will only watch the
// ServicePort number passed in. If the SidecarUrl is not empty string, then
// then we will call that address to get initial state on bootstrap.
func NewSidecarRing(sidecarUrl string, svcName string, svcPort int64) (*SidecarRing, error) {
	ringMgr := NewHashRingManager([]string{})
	looper := director.NewFreeLooper(director.FOREVER, nil)
	go ringMgr.Run(looper)

	scRing := &SidecarRing{
		Manager:       ringMgr,
		managerLooper: looper,
		sidecarUrl:    sidecarUrl,
		svcName:       svcName,
		svcPort:       svcPort,
	}

	// Set up the receiver for incoming requests
	rcvr := receiver.NewReceiver(DefaultReceiverCapacity, scRing.onUpdate)
	// Subscribe to only the service requested
	rcvr.Subscribe(svcName)
	scRing.rcvr = rcvr

	// If we were given a Sidecar address to bootstrap from, then do it. Otherwie
	// we just wait for updates.
	if sidecarUrl != "" {
		err := rcvr.FetchInitialState(sidecarUrl)
		if err != nil {
			return nil, err
		}
	}

	go rcvr.ProcessUpdates()

	return scRing, nil
}

// onUpdate takes care of incoming updates from the receiver
func (r *SidecarRing) onUpdate(state *catalog.ServicesState) {
	newNodes := make(map[string]struct{}, len(r.nodes)+5) // Likely to be similar length

	state.EachService(func(hostname *string, serviceId *string, svc *service.Service) {
		if svc.Name == r.svcName {
			key, err := r.keyForService(svc)
			if err != nil {
				log.Error(err)
				return
			}
			newNodes[key] = struct{}{}
		}
	})

	// Was it it in the new group and not in the old one? Add it.
	for name := range newNodes {
		if _, ok := r.nodes[name]; !ok {
			r.Manager.AddNode(name)
		}
	}

	// In the old group but not in the new one? Remove it.
	for name := range r.nodes {
		if _, ok := newNodes[name]; !ok {
			r.Manager.RemoveNode(name)
		}
	}

	// Overwrite the old set
	r.nodes = newNodes
}

// keyForService takes a service and returns the key we use to store it in the
// hashring. Currently based on the IP address and service port.
func (r *SidecarRing) keyForService(svc *service.Service) (string, error) {
	var matched *service.Port
	for _, port := range svc.Ports {
		if port.ServicePort == r.svcPort {
			matched = &port
			break
		}
	}

	if matched == nil {
		return "", fmt.Errorf(
			"Can't match service port %d for incoming service %s!",
			r.svcPort, svc.ID,
		)
	}

	var key string
	if matched.IP == "" {
		key = svc.Hostname
	} else {
		key = matched.IP
	}

	return fmt.Sprintf("%s:%d", key, matched.Port), nil
}

// HttpListNodesHandler is an http.Handler that will return a JSON-encoded list of
// the Sidecar nodes in the current ring.
func (r *SidecarRing) HttpListNodesHandler(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()

	jsonBytes, err := json.MarshalIndent(&r.nodes, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Write(jsonBytes)
}

// HttpGetNodeHandler is an http.Handler that will return an object containing the
// node that currently owns a specific key.
func (r *SidecarRing) HttpGetNodeHandler(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()

	key := req.FormValue("key")
	if key == "" {
		http.Error(w, `{"status": "error", "message": "Invalid key"}`, 404)
		return
	}

	if r == nil {
		http.Error(w, `{"status": "error", "message": "SidecarRing was nil"}`, 500)
		return
	}

	node, _ := r.Manager.GetNode(key)

	respObj := struct {
		Node string
		Key  string
	}{node, key}

	jsonBytes, err := json.MarshalIndent(respObj, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), 500)
	}

	w.Write(jsonBytes)
}

// HttpMux returns an http.ServeMux configured to run the HTTP handlers on the
// SidecarRing. You can either use this one, or mount the handlers on a mux of your
// own choosing (e.g. Gorilla mux or httprouter)
func (r *SidecarRing) HttpMux() *http.ServeMux {
	updateHandler := func(w http.ResponseWriter, req *http.Request) {
		receiver.UpdateHandler(w, req, r.rcvr)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/nodes/get", r.HttpGetNodeHandler)
	mux.HandleFunc("/nodes", r.HttpListNodesHandler)
	mux.HandleFunc("/update", updateHandler)
	return mux
}

// Shutdown stops the Receiver and the HashringManager
func (r *SidecarRing) Shutdown() {
	r.rcvr.Looper.Quit()
	r.managerLooper.Quit()
}
