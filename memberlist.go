package ringman

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Nitro/memberlist"
	"github.com/relistan/go-director"
	log "github.com/sirupsen/logrus"
)

// A MemberlistRing is a ring backed by Hashicorp's Memberlist directly. It
// exchanges gossip messages directly between instances of this service and
// requires some open ports for them to communicate with each other. The nodes
// will need to have some seeds provided that allow them to find each other.
type MemberlistRing struct {
	Memberlist    *memberlist.Memberlist
	Manager       *HashRingManager
	managerLooper director.Looper
}

// NewDefaultMemberlistRing returns a MemberlistRing configured using the
// DefaultLANConfig from the memberlist documentation. clusterSeeds must be 0 or
// more hosts to seed the cluster with. Note that the ring will be _running_
// when returned from this method.
func NewDefaultMemberlistRing(clusterSeeds []string, port string) (*MemberlistRing, error) {
	return NewMemberlistRing(memberlist.DefaultLANConfig(), clusterSeeds, port, "default")
}

// NewMemberlistRing configures a MemberlistRing according to the Memberlist
// configuration specified. clusterSeeds must be 0 or more hosts to seed the
// cluster with. Note that the ring will be _running_  when returned from this
// method.
//
// * mlConfig is a memberlist config struct
// * clusterSeeds are the hostnames of the machines we'll bootstrap from
// * port is our own service port that the service (not memberist) will use
//
func NewMemberlistRing(mlConfig *memberlist.Config, clusterSeeds []string, port string,
	clusterName string) (*MemberlistRing, error) {

	if clusterSeeds == nil {
		clusterSeeds = []string{}
	}

	if mlConfig.LogOutput == nil {
		mlConfig.LogOutput = &LoggingBridge{}
	}

	mlConfig.ClusterName = clusterName

	// We need to set up the delegate first, so we join the ring with
	// meta-data (otherwise our service port gets skipped over). We'll give
	// it a real ring manager a few lines down.
	delegate := NewDelegate(nil, &NodeMetadata{ServicePort: port})
	mlConfig.Delegate = delegate
	mlConfig.Events = delegate

	list, err := memberlist.Create(mlConfig)
	if err != nil {
		return nil, fmt.Errorf("Unable to create Memberlist cluster: %s", err)
	}

	ringMgr := NewHashRingManager([]string{})
	looper := director.NewFreeLooper(director.FOREVER, nil)
	go ringMgr.Run(looper)

	// Wait for the RingManager to be ready before proceeding
	if !ringMgr.Ping() {
		return nil, fmt.Errorf("Unable to initialize the HashRingManager")
	}

	delegate.RingMan = ringMgr

	// Make sure we have all the nodes added, using the callback in
	// the delegate, which does the right thing.
	for _, node := range list.Members() {
		delegate.NotifyJoin(node)
	}

	_, err = list.Join(clusterSeeds)
	if err != nil {
		return nil, fmt.Errorf("Unable to join Memberlist cluster: %s", err)
	}

	return &MemberlistRing{
		Memberlist:    list,
		Manager:       ringMgr,
		managerLooper: looper,
	}, nil
}

// HttpListNodesHandler is an http.Handler that will return a JSON-encoded list of
// the Memberlist nodes in the current ring.
func (r *MemberlistRing) HttpListNodesHandler(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()

	list := r.Memberlist.Members()

	jsonBytes, err := json.MarshalIndent(&list, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Write(jsonBytes)
}

// HttpGetNodeHandler is an http.Handler that will return an object containing the
// node that currently owns a specific key.
func (r *MemberlistRing) HttpGetNodeHandler(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()

	key := req.FormValue("key")
	if key == "" {
		http.Error(w, `{"status": "error", "message": "Invalid key"}`, 404)
		return
	}

	if r == nil {
		http.Error(w, `{"status": "error", "message": "MemberlistRing was nil"}`, 500)
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
// MemberlistRing. You can either use this one, or mount the handlers on a mux of your
// own choosing (e.g. Gorilla mux or httprouter)
func (r *MemberlistRing) HttpMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/nodes/get", r.HttpGetNodeHandler)
	mux.HandleFunc("/nodes", r.HttpListNodesHandler)
	return mux
}

// Shutdown shuts down the memberlist node and stops the HashRingManager
func (r *MemberlistRing) Shutdown() {
	err := r.Memberlist.Leave(2 * time.Second) // 2 second timeout
	if err != nil {
		log.Debugf("Failed to leave Memberlist cluster: %s", err)
	}

	err = r.Memberlist.Shutdown()
	if err != nil {
		log.Debugf("Failed to shutdown Memberlist: %s", err)
	}

	r.Manager.Stop()

	r.managerLooper.Quit()
}
