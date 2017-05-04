package ringman

import (
	"encoding/json"

	"github.com/Nitro/memberlist"
	log "github.com/Sirupsen/logrus"
)

type Delegate struct {
	ringMan *HashRingManager
}

func NewDelegate(ringMan *HashRingManager) *Delegate {
	delegate := Delegate{
		ringMan: ringMan,
	}

	return &delegate
}

func (d *Delegate) NodeMeta(limit int) []byte {
	if d.ringMan.OurMetadata == nil {
		return []byte("{}")
	}

	data, err := json.Marshal(d.ringMan.OurMetadata)
	if err != nil {
		log.Error("Error encoding Node metadata!")
		data = []byte("{}")
	}
	log.Debugf("Setting metadata to: %s", string(data))

	return data
}

func (d *Delegate) NotifyMsg(message []byte) {
	log.Debugf("NotifyMsg(): %s", string(message))
}

func (d *Delegate) GetBroadcasts(overhead, limit int) [][]byte {
	//log.Debugf("GetBroadcasts(): %d %d", overhead, limit)
	return [][]byte{}
}

func (d *Delegate) LocalState(join bool) []byte {
	log.Debugf("LocalState(): %b", join)
	return []byte{}
}

func (d *Delegate) MergeRemoteState(buf []byte, join bool) {
	log.Debugf("MergeRemoteState(): %s %t", string(buf), join)
}

func (d *Delegate) NotifyJoin(node *memberlist.Node) {
	log.Debugf("NotifyJoin(): %s %s", node.Name, string(node.Meta))
	d.ringMan.AddNode(node.Name)
	d.updateMetadata(node)
}

func (d *Delegate) NotifyLeave(node *memberlist.Node) {
	log.Debugf("NotifyLeave(): %s", node.Name)
	d.ringMan.RemoveNode(node.Name)
}

func (d *Delegate) NotifyUpdate(node *memberlist.Node) {
	log.Debugf("NotifyUpdate(): %s - %s", node.Name, node.Meta)
	d.updateMetadata(node)
}

// updateMetadata decodes the node metadata and tells the ring manager
// about the update. This usually comes from a node joining the cluster
// or sending an update message (NotifyJoin or NotifyUpdate).
func (d *Delegate) updateMetadata(node *memberlist.Node) {
	var meta RingMetadata

	err := json.Unmarshal(node.Meta, &meta)
	if err != nil {
		log.Errorf("Unable to decode node metadata: %s", err)
		return
	}

	d.ringMan.UpdateMetadata(node.Name, &meta)
}
