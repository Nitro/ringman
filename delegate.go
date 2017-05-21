package ringman

import (
	"encoding/json"

	"github.com/hashicorp/memberlist"
	log "github.com/Sirupsen/logrus"
)

type NodeMetadata struct {
	ServicePort string
}

type Delegate struct {
	RingMan      *HashRingManager
	nodeMetadata *NodeMetadata
}

func NewDelegate(ringMan *HashRingManager, meta *NodeMetadata) *Delegate {
	delegate := Delegate{
		RingMan:      ringMan,
		nodeMetadata: meta,
	}

	return &delegate
}

func (d *Delegate) NodeMeta(limit int) []byte {
	data, err := json.Marshal(d.nodeMetadata)
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
	log.Debugf("LocalState(): %t", join)
	return []byte{}
}

func (d *Delegate) MergeRemoteState(buf []byte, join bool) {
	log.Debugf("MergeRemoteState(): %s %t", string(buf), join)
}

func (d *Delegate) NotifyJoin(node *memberlist.Node) {
	log.Debugf("NotifyJoin(): %s %s", node.Name, string(node.Meta))

	if d.RingMan == nil {
		log.Warn("Ring manager was nil in delegate!")
		return
	}

	meta, err := DecodeNodeMetadata(node.Meta)
	if err != nil {
		log.Errorf("Unable to decode metadata for %s", node.Name)
		d.RingMan.AddNode(node.Name)
		return
	}
	d.RingMan.AddNode(node.Name + ":" + meta.ServicePort)
}

func (d *Delegate) NotifyLeave(node *memberlist.Node) {
	log.Debugf("NotifyLeave(): %s", node.Name)
	if d.RingMan == nil {
		log.Error("Ring manager was nil in delegate!")
		return
	}
	d.RingMan.RemoveNode(node.Name)
}

func (d *Delegate) NotifyUpdate(node *memberlist.Node) {
	log.Debugf("NotifyUpdate(): %s - %s", node.Name, node.Meta)
}

// DecodeNodeMetadata takes a byte slice and deserializes it
func DecodeNodeMetadata(data []byte) (*NodeMetadata, error) {
	var meta NodeMetadata
	err := json.Unmarshal(data, &meta)
	if err != nil {
		return nil, err
	}

	return &meta, nil
}
